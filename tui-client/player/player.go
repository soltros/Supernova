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
	Queue         []models.Track
	CurrentIndex  int
	CurrentState  State
	currentFFPlay *exec.Cmd
	mu            sync.Mutex
	
	OnStateChange func(state State, track *models.Track)
)

func AddToQueue(track models.Track) {
	mu.Lock()
	defer mu.Unlock()
	Queue = append(Queue, track)
}

func ClearQueue() {
	mu.Lock()
	defer mu.Unlock()
	Queue = []models.Track{}
	CurrentIndex = 0
}

func PlayTrack(index int) {
	mu.Lock()
	if index < 0 || index >= len(Queue) {
		mu.Unlock()
		return
	}
	CurrentIndex = index
	track := Queue[index]
	mu.Unlock()

	Stop()

	streamURL := fmt.Sprintf("%s/stream/%s", api.BaseURL, track.ID)
	headerArg := fmt.Sprintf("Authorization: Bearer %s", api.JWTToken)
	
	currentFFPlay = exec.Command("ffplay", "-headers", headerArg, "-nodisp", "-autoexit", streamURL)
	currentFFPlay.Start()

	CurrentState = Playing
	
	// Wait for process to exit to play next
	go func(cmd *exec.Cmd, idx int) {
		cmd.Wait()
		mu.Lock()
		if CurrentState == Playing && CurrentIndex == idx {
			// Auto play next
			mu.Unlock()
			Next()
			return
		}
		mu.Unlock()
	}(currentFFPlay, CurrentIndex)

	if OnStateChange != nil {
		OnStateChange(CurrentState, &track)
	}
	
	// Update MPRIS if initialized
	UpdateMPRIS(track, CurrentState)
}

func Next() {
	mu.Lock()
	nextIdx := CurrentIndex + 1
	if nextIdx >= len(Queue) {
		mu.Unlock()
		Stop()
		return
	}
	mu.Unlock()
	PlayTrack(nextIdx)
}

func Prev() {
	mu.Lock()
	prevIdx := CurrentIndex - 1
	if prevIdx < 0 {
		prevIdx = 0
	}
	mu.Unlock()
	PlayTrack(prevIdx)
}

func Stop() {
	if currentFFPlay != nil && currentFFPlay.Process != nil {
		currentFFPlay.Process.Kill()
	}
	CurrentState = Stopped
	if OnStateChange != nil {
		OnStateChange(CurrentState, nil)
	}
}

func TogglePause() {
	// Simple stop for now since ffplay doesn't natively pause via headless stdin reliably
	if CurrentState == Playing {
		Stop()
	} else if CurrentState == Stopped && len(Queue) > 0 {
		PlayTrack(CurrentIndex)
	}
}
