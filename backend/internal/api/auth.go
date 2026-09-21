package api

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/soltros/Supernova/internal/authn"
	"github.com/soltros/Supernova/internal/database"
	"github.com/soltros/Supernova/internal/models"
	"golang.org/x/crypto/bcrypt"
)

func getJWTSecret() []byte {
	secret,err:=authn.Secret()
	if err!=nil{log.Fatal(err)}
	return secret
}

type contextKey string
const userIDKey contextKey="user_id"
const sessionIDKey contextKey="session_id"

type AuthRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	InviteCode string `json:"invite_code"`
}
type AuthResponse struct{Token string `json:"token"`;User *models.User `json:"user"`}

type attemptState struct{failures int;first time.Time;blockedUntil time.Time}
type attemptLimiter struct{mu sync.Mutex;entries map[string]attemptState}
var authAttemptLimiter=&attemptLimiter{entries:map[string]attemptState{}}

func authAttemptKey(r *http.Request,username string)string{
	host,_,err:=net.SplitHostPort(r.RemoteAddr);if err!=nil{host=r.RemoteAddr}
	return host+"|"+strings.ToLower(strings.TrimSpace(username))
}
func (l *attemptLimiter) allowed(key string)bool{
	l.mu.Lock();defer l.mu.Unlock()
	now:=time.Now();s:=l.entries[key]
	if now.Before(s.blockedUntil){return false}
	if !s.first.IsZero() && now.Sub(s.first)>10*time.Minute{delete(l.entries,key)}
	return true
}
func (l *attemptLimiter) failed(key string){
	l.mu.Lock();defer l.mu.Unlock()
	now:=time.Now();s:=l.entries[key]
	if s.first.IsZero() || now.Sub(s.first)>10*time.Minute{s=attemptState{first:now}}
	s.failures++
	if s.failures>=5{s.blockedUntil=now.Add(5*time.Minute);s.failures=0;s.first=now}
	l.entries[key]=s
}
func (l *attemptLimiter) success(key string){l.mu.Lock();delete(l.entries,key);l.mu.Unlock()}

func (s *Server) handleRegister()http.HandlerFunc{return func(w http.ResponseWriter,r *http.Request){
	var req AuthRequest
	if err:=json.NewDecoder(r.Body).Decode(&req);err!=nil{http.Error(w,"invalid request body",400);return}
	req.Username=strings.TrimSpace(req.Username)
	if len(req.Username)<3||len(req.Username)>64||len(req.Password)<6||len(req.Password)>72{http.Error(w,"username must be 3–64 bytes and password 6–72 bytes",400);return}
	key:=authAttemptKey(r,req.Username)
	if !authAttemptLimiter.allowed(key){http.Error(w,"too many attempts; try again later",http.StatusTooManyRequests);return}
	if err:=s.repo.RegistrationAllowed(r.Context(),req.InviteCode,os.Getenv("REGISTRATION_INVITE_CODE"));err!=nil{
		authAttemptLimiter.failed(key)
		if errors.Is(err,database.ErrInviteRequired){http.Error(w,err.Error(),http.StatusForbidden)}else{http.Error(w,"registration unavailable",500)}
		return
	}
	hash,err:=bcrypt.GenerateFromPassword([]byte(req.Password),bcrypt.DefaultCost)
	if err!=nil{http.Error(w,"failed to hash password",500);return}
	user,err:=s.repo.RegisterUser(r.Context(),req.Username,string(hash),req.InviteCode,os.Getenv("REGISTRATION_INVITE_CODE"))
	if err!=nil{authAttemptLimiter.failed(key);if errors.Is(err,database.ErrInviteRequired){http.Error(w,err.Error(),403)}else{http.Error(w,"username already exists",409)};return}
	if enc,err:=EncryptPassword(req.Password,getJWTSecret());err==nil{_ = s.repo.SetSubsonicPassword(r.Context(),user.Username,enc)}
	token,_,err:=s.issueSessionToken(r.Context(),user.ID)
	if err!=nil{http.Error(w,"failed to generate token",500);return}
	authAttemptLimiter.success(key)
	json.NewEncoder(w).Encode(AuthResponse{Token:token,User:user})
}}

func (s *Server) handleLogin()http.HandlerFunc{return func(w http.ResponseWriter,r *http.Request){
	var req AuthRequest
	if err:=json.NewDecoder(r.Body).Decode(&req);err!=nil{http.Error(w,"invalid request body",400);return}
	req.Username=strings.TrimSpace(req.Username)
	key:=authAttemptKey(r,req.Username)
	if !authAttemptLimiter.allowed(key){http.Error(w,"too many attempts; try again later",http.StatusTooManyRequests);return}
	user,hash,err:=s.repo.GetUserByUsername(r.Context(),req.Username)
	if err!=nil{http.Error(w,"database error",500);return}
	if user==nil || bcrypt.CompareHashAndPassword([]byte(hash),[]byte(req.Password))!=nil{
		authAttemptLimiter.failed(key);http.Error(w,"invalid username or password",401);return
	}
	if enc,err:=EncryptPassword(req.Password,getJWTSecret());err==nil{_ = s.repo.SetSubsonicPassword(r.Context(),user.Username,enc)}
	token,_,err:=s.issueSessionToken(r.Context(),user.ID)
	if err!=nil{http.Error(w,"failed to generate token",500);return}
	authAttemptLimiter.success(key)
	json.NewEncoder(w).Encode(AuthResponse{Token:token,User:user})
}}

func (s *Server) issueSessionToken(ctx context.Context,userID string)(string,string,error){
	expires:=time.Now().Add(7*24*time.Hour)
	sessionID,err:=s.repo.CreateSession(ctx,userID,expires);if err!=nil{return "","",err}
	token:=jwt.NewWithClaims(jwt.SigningMethodHS256,jwt.MapClaims{"user_id":userID,"jti":sessionID,"kind":"session","exp":expires.Unix()})
	signed,err:=token.SignedString(getJWTSecret());if err!=nil{return "","",err}
	return signed,sessionID,nil
}

func generateMediaTicket(userID,sessionID,scope,resource string)(string,error){
	token:=jwt.NewWithClaims(jwt.SigningMethodHS256,jwt.MapClaims{
		"user_id":userID,"jti":sessionID,"kind":"media","scope":scope,"resource":resource,
		"exp":time.Now().Add(2*time.Minute).Unix(),
	})
	return token.SignedString(getJWTSecret())
}

func (s *Server) handleLogout()http.HandlerFunc{return func(w http.ResponseWriter,r *http.Request){
	userID:=r.Context().Value(userIDKey).(string)
	sessionID:=r.Context().Value(sessionIDKey).(string)
	if err:=s.repo.RevokeSession(r.Context(),sessionID,userID);err!=nil{http.Error(w,"failed to revoke session",500);return}
	w.WriteHeader(http.StatusNoContent)
}}

func (s *Server) handleChangePassword()http.HandlerFunc{return func(w http.ResponseWriter,r *http.Request){
	user:=r.Context().Value(contextKey("user")).(*models.User)
	sessionID:=r.Context().Value(sessionIDKey).(string)
	var req struct{Current string `json:"current_password"`;New string `json:"new_password"`}
	if err:=json.NewDecoder(r.Body).Decode(&req);err!=nil{http.Error(w,"invalid request body",400);return}
	if len(req.New)<6||len(req.New)>72{http.Error(w,"new password must be 6–72 bytes",400);return}
	_,hash,err:=s.repo.GetUserByUsername(r.Context(),user.Username)
	if err!=nil{http.Error(w,"database error",500);return}
	if bcrypt.CompareHashAndPassword([]byte(hash),[]byte(req.Current))!=nil{http.Error(w,"current password is incorrect",401);return}
	newHash,err:=bcrypt.GenerateFromPassword([]byte(req.New),bcrypt.DefaultCost);if err!=nil{http.Error(w,"failed to hash password",500);return}
	if err:=s.repo.UpdatePassword(r.Context(),user.ID,string(newHash));err!=nil{http.Error(w,"failed to update password",500);return}
	if enc,err:=EncryptPassword(req.New,getJWTSecret());err==nil{_ = s.repo.SetSubsonicPassword(r.Context(),user.Username,enc)}
	if err:=s.repo.RevokeOtherSessions(r.Context(),user.ID,sessionID);err!=nil{http.Error(w,"failed to revoke other sessions",500);return}
	w.WriteHeader(http.StatusNoContent)
}}

func (s *Server) handleMediaTicket()http.HandlerFunc{return func(w http.ResponseWriter,r *http.Request){
	userID:=r.Context().Value(userIDKey).(string)
	sessionID:=r.Context().Value(sessionIDKey).(string)
	var req struct{Scope string `json:"scope"`;Resource string `json:"resource"`}
	if err:=json.NewDecoder(r.Body).Decode(&req);err!=nil{http.Error(w,"invalid request body",400);return}
	switch req.Scope{case "stream","download-track","download-album":default:http.Error(w,"invalid media scope",400);return}
	if strings.TrimSpace(req.Resource)==""{http.Error(w,"resource is required",400);return}
	ticket,err:=generateMediaTicket(userID,sessionID,req.Scope,req.Resource);if err!=nil{http.Error(w,"failed to issue media ticket",500);return}
	json.NewEncoder(w).Encode(map[string]string{"ticket":ticket})
}}

func (s *Server) requireAuth(next http.HandlerFunc)http.HandlerFunc{return func(w http.ResponseWriter,r *http.Request){
	allowMedia:=(r.Method==http.MethodGet||r.Method==http.MethodHead)&&(strings.HasPrefix(r.URL.Path,"/api/stream/")||strings.HasPrefix(r.URL.Path,"/api/download/"))
	user,sessionID,err:=authn.Authenticate(r,s.repo,allowMedia)
	if err!=nil{http.Error(w,"unauthorized",401);return}
	ctx:=context.WithValue(r.Context(),userIDKey,user.ID)
	ctx=context.WithValue(ctx,sessionIDKey,sessionID)
	ctx=context.WithValue(ctx,contextKey("user"),user)
	next(w,r.WithContext(ctx))
}}

func (s *Server) requireAdmin(next http.HandlerFunc)http.HandlerFunc{return s.requireAuth(func(w http.ResponseWriter,r *http.Request){
	user:=r.Context().Value(contextKey("user")).(*models.User)
	if !user.IsAdmin{http.Error(w,"administrator access required",403);return}
	next(w,r)
})}
