package authn

import (
	"errors"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/soltros/Supernova/internal/database"
	"github.com/soltros/Supernova/internal/models"
)

func Secret() ([]byte,error){
	secret:=[]byte(os.Getenv("JWT_SECRET"))
	if len(secret)<32{return nil,errors.New("JWT_SECRET must contain at least 32 characters")}
	return secret,nil
}

func parseToken(value string)(jwt.MapClaims,error){
	secret,err:=Secret();if err!=nil{return nil,err}
	token,err:=jwt.Parse(value,func(*jwt.Token)(any,error){return secret,nil},
		jwt.WithValidMethods([]string{"HS256"}),jwt.WithExpirationRequired())
	if err!=nil || !token.Valid{return nil,errors.New("invalid token")}
	claims,ok:=token.Claims.(jwt.MapClaims);if !ok{return nil,errors.New("invalid claims")}
	return claims,nil
}

func mediaScopeForRequest(r *http.Request)(scope,resource string,ok bool){
	path:=r.URL.Path
	switch{
	case strings.HasPrefix(path,"/api/stream/"):
		return "stream",strings.TrimPrefix(path,"/api/stream/"),true
	case strings.HasPrefix(path,"/api/download/track/"):
		return "download-track",strings.TrimPrefix(path,"/api/download/track/"),true
	case strings.HasPrefix(path,"/api/download/album/"):
		return "download-album",strings.TrimPrefix(path,"/api/download/album/"),true
	default:
		return "","",false
	}
}

func Authenticate(r *http.Request,repo *database.Repository,allowMediaTicket bool)(*models.User,string,error){
	header:=r.Header.Get("Authorization")
	if header!=""{
		fields:=strings.Fields(header)
		if len(fields)!=2 || !strings.EqualFold(fields[0],"Bearer"){return nil,"",errors.New("invalid authorization header")}
		claims,err:=parseToken(fields[1]);if err!=nil{return nil,"",err}
		userID,_:=claims["user_id"].(string)
		sessionID,_:=claims["jti"].(string)
		kind,_:=claims["kind"].(string)
		if userID=="" || sessionID=="" || kind=="media"{return nil,"",errors.New("invalid session token")}
		if !repo.SessionActive(r.Context(),sessionID,userID){return nil,"",errors.New("session revoked or expired")}
		user,err:=repo.GetUserByID(r.Context(),userID)
		if err!=nil || user==nil{return nil,"",errors.New("user not found")}
		return user,sessionID,nil
	}
	if !allowMediaTicket{return nil,"",errors.New("missing token")}
	ticket:=r.URL.Query().Get("ticket")
	if ticket==""{return nil,"",errors.New("missing media ticket")}
	claims,err:=parseToken(ticket);if err!=nil{return nil,"",err}
	if kind,_:=claims["kind"].(string);kind!="media"{return nil,"",errors.New("not a media ticket")}
	userID,_:=claims["user_id"].(string)
	sessionID,_:=claims["jti"].(string)
	scope,_:=claims["scope"].(string)
	resource,_:=claims["resource"].(string)
	wantScope,wantResource,ok:=mediaScopeForRequest(r)
	if !ok || scope!=wantScope || resource!=wantResource{return nil,"",errors.New("media ticket scope mismatch")}
	if !repo.SessionActive(r.Context(),sessionID,userID){return nil,"",errors.New("session revoked or expired")}
	user,err:=repo.GetUserByID(r.Context(),userID)
	if err!=nil || user==nil{return nil,"",errors.New("user not found")}
	return user,sessionID,nil
}
