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
		sendError(w, "server error getting houses", 500)
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
	houseID, err := getInt32Param(req, "houseID")
	if err != nil {
		sendError(w, err.Error(), 400)
		return
	}

	exists, err := h.store.GetHouseExistsByHouseID(houseID)
	if err != nil {
		log.Println(err)
		sendError(w, "server error getting House", 500)
		return
	}
	if !exists {
		sendError(w, fmt.Sprintf("no House exists with ID %v", houseID), 404)
		return
	}

	trees, err := h.store.GetTreesByHouseID(houseID)
	if err != nil {
		log.Println(err)
		sendError(w, "server error getting Trees", 500)
		return
	}

	if len(trees) == 0 {
		sendError(w, fmt.Sprintf("no trees growing at house %v", houseID), 404)
		return
	}

	sendJSON(w, trees)
}

// AddHouse plants a new tree on location at a given house.
func (h *handler) AddHouse(w http.ResponseWriter, req *http.Request) {
	var house House
	err := json.NewDecoder(req.Body).Decode(&house)
	if err != nil {
		log.Println(err)
		sendError(w, "error decoding request body as House", 400)
		return
	}

	if house.AddressOne == "" {
		sendError(w, "must specify address one", 400)
		return
	}
	if house.City == "" {
		sendError(w, "must specify city", 400)
		return
	}
	if house.State == "" {
		sendError(w, "must specify state", 400)
		return
	}
	if house.State == "" {
		sendError(w, "must specify zip", 400)
		return
	}

	err = h.store.AddHouse(&house)
	if err != nil {
		log.Println(err)
		sendError(w, "server error adding House", 500)
		return
	}

	sendJSON(w, house)
}

// AddTreeByHouseID plants a new tree on location at a given house.
func (h *handler) AddTreeByHouseID(w http.ResponseWriter, req *http.Request) {
	houseID, err := getInt32Param(req, "houseID")
	if err != nil {
		sendError(w, err.Error(), 400)
		return
	}

	exists, err := h.store.GetHouseExistsByHouseID(houseID)
	if err != nil {
		log.Println(err)
		sendError(w, "server error getting House", 500)
		return
	}
	if !exists {
		sendError(w, fmt.Sprintf("no House exists with ID %v", houseID), 404)
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
		sendError(w, "must specify species", 400)
		return
	}
	if tree.XCoord <= 0 || tree.XCoord > 255 {
		sendError(w, "must specify an x coordinate between 1-255", 400)
		return
	}
	if tree.YCoord <= 0 || tree.YCoord > 255 {
		sendError(w, "must specify a y coordinate between 1-255", 400)
		return
	}

	err = h.store.AddTreeByHouseID(&tree, houseID)
	if err != nil {
		log.Println(err)
		if errors.Is(err, ErrDuplicateTree) {
			sendError(w, err.Error(), 400)
			return
		}
		sendError(w, "server error adding Tree", 500)
		return
	}

	sendJSON(w, tree)
}

// SendStormByHouseID brings down a random tree at a given home site.
func (h *handler) SendStormByHouseID(w http.ResponseWriter, req *http.Request) {
	houseID, err := getInt32Param(req, "houseID")
	if err != nil {
		sendError(w, err.Error(), 400)
		return
	}

	exists, err := h.store.GetHouseExistsByHouseID(houseID)
	if err != nil {
		log.Println(err)
		sendError(w, "server error getting House", 500)
		return
	}
	if !exists {
		sendError(w, fmt.Sprintf("no House exists with ID %v", houseID), 404)
		return
	}

	err = h.store.FellRandomTreeByHouseID(houseID)
	if err != nil {
		log.Println(err)
		if errors.Is(err, ErrNoTrees) {
			sendError(w, err.Error(), 400)
			return
		}
		sendError(w, "server error sending storm", 500)
		return
	}

	trees, err := h.store.GetTreesByHouseID(houseID)
	if err != nil {
		sendError(w, "server error getting trees post-storm", 500)
		return
	}

	sendJSON(w, trees)
}

// RemoveTreeByTreeID removes a fallen tree or else complains that it's still thriving.
func (h *handler) RemoveTreeByTreeID(w http.ResponseWriter, req *http.Request) {
	treeID, err := getInt32Param(req, "treeID")
	if err != nil {
		sendError(w, err.Error(), 400)
		return
	}

	fallen, err := h.store.GetTreeFallenByTreeID(treeID)
	if err != nil {
		log.Println(err)
		if errors.Is(err, ErrNoMatchingRecord) {
			sendError(w, fmt.Sprintf("no Tree found with ID %v", treeID), 404)
			return
		}
		sendError(w, "server error getting Tree", 500)
		return
	}

	if !fallen {
		sendError(w, "call us back when the Tree falls", 400)
		return
	}

	err = h.store.RemoveTreeByTreeID(treeID)
	if err != nil {
		sendError(w, "server error removing Tree", 500)
		return
	}

	w.WriteHeader(200)
}

func getInt32Param(req *http.Request, key string) (int32, error) {
	vars := mux.Vars(req)
	paramStr, _ := vars[key]
	paramInt, err := strconv.ParseInt(paramStr, 10, 32)

	if err != nil || paramInt == 0 {
		return 0, errors.New("param %s must be a valid, non-zero numeral")
	}

	return int32(paramInt), nil
}

func sendError(w http.ResponseWriter, msg string, status int) {
	err := ErrorResponse{
		Message: msg,
	}
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(err)
}

func sendJSON(w http.ResponseWriter, object interface{}) {
	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(object)
}
