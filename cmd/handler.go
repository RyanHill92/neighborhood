package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

type handler struct {
	store *MySQLStore
}

// HealthReport offers a deep look into the intricacies of app health.
type HealthReport struct {
	Status string `json:"status"`
}

// ErrorResponse reports an error.
type ErrorResponse struct {
	Message string `json:"message"`
}

// HelloHandler says hello.
func (h *handler) ReportHealth(w http.ResponseWriter, req *http.Request) {
	report := HealthReport{Status: "so healthy right now!"}
	sendJSON(w, report)
}

// GetAllHouses returns all houses in the neighborhood.
func (h *handler) GetAllHouses(w http.ResponseWriter, req *http.Request) {
	houses, err := h.store.GetAllHouses()
	if err != nil {
		log.Println(err)
		sendError(w, "server error", 500)
		return
	}

	if len(houses) == 0 {
		sendError(w, "no houses in the neighborhood", 404)
		return
	}

	sendJSON(w, houses)
}

// GetTreesByHouseID lists all trees growing on-site at a specific house.
func (h *handler) GetTreesByHouseID(w http.ResponseWriter, req *http.Request) {
	houseID, err := getHouseID(req)
	if err != nil {
		sendError(w, err.Error(), 400)
		return
	}

	trees, err := h.store.GetTreesByHouseID(houseID)
	if err != nil {
		log.Println(err)
		sendError(w, "server error", 500)
		return
	}

	if len(trees) == 0 {
		sendError(w, fmt.Sprintf("no trees found for house %v", houseID), 404)
		return
	}

	sendJSON(w, trees)
}

func (h *handler) AddTreeByHouseID(w http.ResponseWriter, req *http.Request) {
	houseID, err := getHouseID(req)
	if err != nil {
		sendError(w, err.Error(), 400)
		return
	}

	var tree Tree
	err = json.NewDecoder(req.Body).Decode(&tree)
	if err != nil {
		log.Println(err)
		sendError(w, "error decoding request body as Tree", 400)
		return
	}

	if tree.Species == "" {
		sendError(w, "must specify a non-empty species", 400)
	}
	if tree.XCoord <= 0 {
		sendError(w, "must specify a positive, non-zero x coordinate", 400)
	}
	if tree.YCoord <= 0 {
		sendError(w, "must specify a positive, non-zero y coordinate", 400)
	}

	err = h.store.AddTreeByHouseID(&tree, houseID)
	if err != nil {
		log.Println(err)
		msg := "server error"
		if errors.Is(err, ErrDuplicateTree) {
			msg = err.Error()
		}
		sendError(w, msg, 500)
		return
	}

	sendJSON(w, tree)
}

func getHouseID(req *http.Request) (int32, error) {
	vars := mux.Vars(req)
	houseIDStr, _ := vars["houseID"]
	houseID, err := strconv.ParseInt(houseIDStr, 10, 32)

	if err != nil || houseID == 0 {
		return 0, errors.New("path must include a valid, non-zero house ID")
	}

	return int32(houseID), nil
}

func sendError(w http.ResponseWriter, msg string, status int) {
	err := ErrorResponse{
		Message: msg,
	}
	w.WriteHeader(status)
	sendJSON(w, err)
}

func sendJSON(w http.ResponseWriter, object interface{}) {
	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(object)
	return
}

// GET Trees by House ID
// GET House by ID
// POST Tree by House ID
// POST House
// POST Storm by ???
