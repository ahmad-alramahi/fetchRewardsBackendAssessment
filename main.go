package main

import (
	"encoding/json"
	"github.com/google/uuid"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

type processedReceipt struct {
	Id             string `json:"id"`
	Retailer       string `json:"retailer"`
	PurchaseDay    int    `json:"purchaseDate"`
	PurchaseHour   int    `json:"purchaseHour"`
	PurchaseMinute int    `json:"purchaseMinute"`
	Total          string `json:"total"`
	Items          []item `json:"items"`
	Points         int    `json:"points"`
}

type rawReceipt struct {
	Retailer     string `json:"retailer"`
	PurchaseDate string `json:"purchaseDate"`
	PurchaseTime string `json:"purchaseTime"`
	Total        string `json:"total"`
	Items        []item `json:"items"`
}
type item struct {
	ShortDescription string `json:"shortDescription"`
	Price            string `json:"price"`
}

type idResponse struct {
	Id string `json:"id"`
}

type pointsResponse struct {
	Points int `json:"points"`
	exists bool
}

var idsToPoints = make(map[string]pointsResponse)

func ProcessReceipt(w http.ResponseWriter, req *http.Request) {
	var incomingReceipt rawReceipt

	if req.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	jsonDecodeErr := json.NewDecoder(req.Body).Decode(&incomingReceipt)
	if jsonDecodeErr != nil {
		http.Error(w, jsonDecodeErr.Error(), http.StatusBadRequest)
	}
	var toBeProcessedReceipt processedReceipt
	parsedDay, _ := strconv.Atoi(strings.Split(incomingReceipt.PurchaseDate, "-")[2])
	splitTime := strings.Split(incomingReceipt.PurchaseTime, ":")
	parsedHour, _ := strconv.Atoi(splitTime[0])
	parsedMinute, _ := strconv.Atoi(splitTime[1])

	toBeProcessedReceipt.Id = uuid.New().String()
	toBeProcessedReceipt.Retailer = incomingReceipt.Retailer
	toBeProcessedReceipt.Items = incomingReceipt.Items
	toBeProcessedReceipt.PurchaseHour = parsedHour
	toBeProcessedReceipt.PurchaseMinute = parsedMinute
	toBeProcessedReceipt.PurchaseDay = parsedDay
	toBeProcessedReceipt.Total = incomingReceipt.Total
	parsePoints(&toBeProcessedReceipt)

	var idOut idResponse
	idOut.Id = toBeProcessedReceipt.Id
	var pointsToBeSaved pointsResponse
	pointsToBeSaved.Points = toBeProcessedReceipt.Points
	pointsToBeSaved.exists = true
	idsToPoints[toBeProcessedReceipt.Id] = pointsToBeSaved

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	jsonEncodeErr := json.NewEncoder(w).Encode(idOut)
	if jsonEncodeErr != nil {
		http.Error(w, jsonEncodeErr.Error(), http.StatusBadRequest)
		return
	}
}

func parsePoints(receipt *processedReceipt) {
	receipt.Points = 0
	alphaNumeric := regexp.MustCompile("[a-zA-Z0-9]")
	temp := len(alphaNumeric.FindAllStringIndex(receipt.Retailer, -1))
	receipt.Points += temp
	cents := strings.Split(receipt.Total, ".")[1]
	if cents == "00" {
		receipt.Points += 75
	}
	if cents == "25" || cents == "50" || cents == "75" {
		receipt.Points += 25
	}
	receipt.Points += 5 * (len(receipt.Items) / 2)
	for _, item := range receipt.Items {
		trimLen := len(strings.Trim(item.ShortDescription, " "))
		if trimLen%3 == 0 {
			convertedFloat, _ := strconv.ParseFloat(item.Price, 32)
			receipt.Points += int(math.Ceil(.2 * convertedFloat))
		}
	}
	if receipt.PurchaseDay%2 == 1 {
		receipt.Points += 6
	}
	if receipt.PurchaseHour >= 14 && receipt.PurchaseHour < 16 {
		if receipt.PurchaseHour != 14 || receipt.PurchaseMinute != 0 {
			receipt.Points += 10
		}
	}
}

func getPointsById(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := req.PathValue("id")
	var pointsOut = idsToPoints[id]
	if !pointsOut.exists {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	jsonEncodeErr := json.NewEncoder(w).Encode(pointsOut)
	if jsonEncodeErr != nil {
		http.Error(w, jsonEncodeErr.Error(), http.StatusBadRequest)
		return
	}
}

func main() {
	http.HandleFunc("/receipts/process", ProcessReceipt)
	http.HandleFunc("/receipts/{id}/points", getPointsById)
	err := http.ListenAndServe(":8090", nil)
	if err != nil {
		return
	}
}
