package player

import (
	"fmt"
	"os/exec"
	"sync"
	
	"github.com/soltros/Supernova/tui-client/api"
	"github.com/soltros/Supernova/tui-client/models"
)

type State int

const (
	Stopped State = iota
	Playing
	Paused
)

var (
	queue         []models.Track
	currentIndex  int
	currentState  State
	currentFFPlay *exec.Cmd
	mu            sync.Mutex
	
	OnStateChange func(state State, track *models.Track)
)

func GetQueueLength() int {
	mu.Lock()
	defer mu.Unlock()
	return len(queue)
}

func GetCurrentState() State {
	mu.Lock()
	defer mu.Unlock()
	return currentState
}

func AddToQueue(track models.Track) {
	mu.Lock()
	defer mu.Unlock()
	queue = append(queue, track)
}

func ClearQueue() {
	mu.Lock()
	defer mu.Unlock()
	queue = []models.Track{}
	currentIndex = 0
}

func PlayTrack(index int) {
	mu.Lock()
	if index < 0 || index >= len(queue) {
		mu.Unlock()
		return
	}
	currentIndex = index
	track := queue[index]
	mu.Unlock()

	Stop()

	streamURL := fmt.Sprintf("%s/stream/%s", api.BaseURL, track.ID)
	headerArg := fmt.Sprintf("Authorization: Bearer %s", api.JWTToken)
	
	cmd := exec.Command("ffplay", "-headers", headerArg, "-nodisp", "-autoexit", streamURL)
	cmd.Start()

	mu.Lock()
	currentFFPlay = cmd
	currentState = Playing
	mu.Unlock()
	
	// Wait for process to exit to play next
	go func(c *exec.Cmd, idx int) {
		c.Wait()
		mu.Lock()
		if currentState == Playing && currentIndex == idx {
			// Auto play next
			mu.Unlock()
			Next()
			return
		}
		mu.Unlock()
	}(cmd, index)

	if OnStateChange != nil {
		OnStateChange(Playing, &track)
	}
	
	// Update MPRIS if initialized
	UpdateMPRIS(track, Playing)
}

func Next() {
	mu.Lock()
	nextIdx := currentIndex + 1
	if nextIdx >= len(queue) {
		mu.Unlock()
		Stop()
		return
	}
	mu.Unlock()
	PlayTrack(nextIdx)
}

func Prev() {
	mu.Lock()
	prevIdx := currentIndex - 1
	if prevIdx < 0 {
		prevIdx = 0
	}
	mu.Unlock()
	PlayTrack(prevIdx)
}

func Stop() {
	mu.Lock()
	if currentFFPlay != nil && currentFFPlay.Process != nil {
		currentFFPlay.Process.Kill()
	}
	currentState = Stopped
	mu.Unlock()

	if OnStateChange != nil {
		OnStateChange(Stopped, nil)
	}
}

func TogglePause() {
	mu.Lock()
	state := currentState
	qLen := len(queue)
	idx := currentIndex
	mu.Unlock()

	// Simple stop for now since ffplay doesn't natively pause via headless stdin reliably
	if state == Playing {
		Stop()
	} else if state == Stopped && qLen > 0 {
		PlayTrack(idx)
	}
}
