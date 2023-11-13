package controllers

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/josecleiton/domino/app/game"
	"github.com/josecleiton/domino/app/models"
)

type gameStateRequest struct {
	Player int                `json:"jogador"`
	Hand   []string           `json:"mao"`
	Table  []string           `json:"mesa"`
	Plays  []playStateRequest `json:"jogadas"`
}

type playStateRequest struct {
	Player    int    `json:"jogador"`
	Bone      string `json:"pedra"`
	Direction string `json:"lado"`
}

type playStateResponse struct {
	Player    int     `json:"jogador"`
	Bone      *string `json:"pedra"`
	Direction *string `json:"lado"`
}

func GameHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	w.Header().Set("Content-Type", "application/json")

	var request gameStateRequest

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&request)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		log.Printf("Error happened in JSON marshal. Err: %s\n", err)

		return
	}

	domino := gameRequestToDomain(&request)

	play, err := game.Play(domino)
	if err != nil {
		const status = http.StatusBadRequest
		errorMap := map[string]interface{}{
			"error":  err.Error(),
			"status": http.StatusText(status),
			"code":   status,
		}

		w.WriteHeader(status)

		jsonResp, marshalErr := json.Marshal(errorMap)
		if marshalErr != nil {
			log.Printf("Error happened in JSON marshal. Err: %s\n", marshalErr)
			w.Write([]byte(err.Error()))
		} else {
			w.Write(jsonResp)
		}

		log.Printf("Error happened in play. Err: %s\n", err)
		return
	}

	resp := dominoPlayToResponse(*play)
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		log.Printf("Error happened in JSON marshal. Err: %s\n", err)
	}

	w.Write(jsonResp)
}

func gameRequestToDomain(request *gameStateRequest) *models.DominoGameState {
	hand := make([]models.Domino, 0, len(request.Hand))
	table := make(map[int]map[int]bool, models.DominoUniqueBones)
	plays := make([]models.DominoPlay, 0, len(request.Plays))

	for _, bone := range request.Hand {
		hand = append(hand, models.DominoFromString(bone))
	}

	for _, bone := range request.Table {
		domino := models.DominoFromString(bone)
		if _, ok := table[domino.X]; !ok {
			table[domino.X] = make(map[int]bool, models.DominoUniqueBones)
		}
		if _, ok := table[domino.Y]; !ok {
			table[domino.Y] = make(map[int]bool, models.DominoUniqueBones)
		}

		table[domino.X][domino.Y] = true
		table[domino.Y][domino.X] = true
	}

	for _, play := range request.Plays {
		plays = append(plays, models.DominoPlay{
			PlayerPosition: play.Player,
			Bone: models.DominoInTable{
				Domino:   models.DominoFromString(play.Bone),
				Reversed: strings.HasPrefix(strings.ToLower(play.Direction), "d"),
			},
		})
	}

	return &models.DominoGameState{
		PlayerPosition: request.Player,
		Hand:           hand,
		Table:          table,
		Plays:          plays,
	}
}

func dominoPlayToResponse(dominoPlay models.DominoPlayWithPass) *playStateResponse {
	if dominoPlay.Pass() {
		return &playStateResponse{Player: dominoPlay.PlayerPosition}
	}

	direction := "esquerda"

	if dominoPlay.Bone.Reversed {
		direction = "direita"
	}

	bone := dominoPlay.Bone.Domino.String()
	return &playStateResponse{
		Player:    dominoPlay.PlayerPosition,
		Bone:      &bone,
		Direction: &direction,
	}
}
