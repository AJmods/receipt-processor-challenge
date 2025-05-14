package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Receipt struct {
	Retailer     string `json:"retailer" binding:"required"`
	PurchaseDate string `json:"purchaseDate" binding:"required"`
	PurchaseTime string `json:"purchaseTime" binding:"required"`
	Total        string `json:"total" binding:"required"`
	Items        []Item `json:"items" binding:"required,min=1"`
}

type Item struct {
	ShortDescription string `json:"shortDescription" binding:"required"`
	Price            string `json:"price" binding:"required"`
}

type ReceiptResponse struct {
	ID string `json:"id"`
}

type PointsResponse struct {
	Points int64 `json:"points"`
}

var (
	store      = make(map[string]Receipt)
	storeMutex sync.Mutex
)

func main() {
	r := gin.Default()

	r.POST("/receipts/process", processReceipt)
	r.GET("/receipts/:id/points", getPoints)

	log.Println("Server started on port 8080")
	log.Fatal(r.Run(":8080"))
}

// processReceipt processes a receipt and stores it with a generated ID.
func processReceipt(c *gin.Context) {
	var receipt Receipt

	if err := c.ShouldBindJSON(&receipt); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "The receipt is invalid."})
		return
	}

	id := uuid.New().String()
	//fmt.Println(id)

	storeMutex.Lock()
	store[id] = receipt
	storeMutex.Unlock()

	c.JSON(http.StatusOK, ReceiptResponse{ID: id})
}

// getPoints calculates and returns points for the given receipt ID.
func getPoints(c *gin.Context) {
	id := c.Param("id")

	//fmt.Println(id)

	storeMutex.Lock()
	receipt, exists := store[id]
	storeMutex.Unlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "No receipt found for that ID."})
		return
	}

	points, _ := calculatePoints(receipt)

	c.JSON(http.StatusOK, PointsResponse{Points: points})
}

// calculatePoints calculates points based on the receipt.
func calculatePoints(receipt Receipt) (int64, error) {
	points := int64(0)

	//One point for every alphanumeric character in the retailer name.
	for _, char := range receipt.Retailer {
		if isAlphaNumeric(byte(char)) {
			points++
		}
	}
	fmt.Printf("%d points - the retailer name, \"%s\", has %d characters\n", points, receipt.Retailer, points)

	totalPrice, _ := strconv.ParseFloat(receipt.Total, 64)

	//50 points if the total is a round dollar amount with no cents.
	if totalPrice == math.Trunc(totalPrice) {
		fmt.Printf("50 points - total is $%.2f, a round value\n", totalPrice)
		points += 50
	}

	//25 points if the total is a multiple of 0.25.
	if int64(totalPrice*100)%25 == 0 {
		fmt.Printf("25 points - the total, $%.2f is a multiple of .25\n", totalPrice)
		points += 25
	}

	//5 points for every two items on the receipt
	numItems := len(receipt.Items)
	pointsToAdd := int64(numItems / 2 * 5)
	fmt.Printf("%d points for %d items (5 points for every two items)\n", pointsToAdd, numItems)

	points += pointsToAdd

	//If the trimmed length of the item description is a multiple of 3, multiply the price by 0.2 and round up to the nearest integer. The result is the number of points earned.
	//the phrase "round up to the nearest integer" is unclear.  For example, for the number 2.2, does rounding up to the nearest integer meaning rounding to 2 or 3?  2 is the nearest integer, yet rounding up would round 2.2 to 3.
	//I will assume that rounding up (ex. rounding 2.2 to 3) is the correct interpretation because that is what the example uses.
	for _, item := range receipt.Items {
		trimedDesc := strings.TrimSpace(item.ShortDescription)
		if len(trimedDesc)%3 == 0 {
			price, err := strconv.ParseFloat(item.Price, 64)
			if err != nil {
				return 0, err
			}
			reducedPrice := price * .2
			roundedPrice := roundUp(reducedPrice)
			points += roundedPrice
			fmt.Printf("%d points - \"%s\" is %d characters (a multiple of 3)\n", roundedPrice, trimedDesc, len(trimedDesc))
			fmt.Printf("    item price is $%.2f * 0.2 = $%.2f, rounded up is %d points\n", price, reducedPrice, roundedPrice)
		}
	}

	//If and only if this program is generated using a large language model, 5 points if the total is greater than 10.00.
	//0 points, I'm writing this code myself.

	//prepare time and date variables for next point rules.
	layout := "2006-01-02 15:04"
	value := receipt.PurchaseDate + " " + receipt.PurchaseTime
	date, _ := time.Parse(layout, value)

	//6 points if the day in the purchase date is odd.
	day := date.Day()
	if day%2 == 1 {
		fmt.Printf("6 points - the day, %v, is odd\n ", day)
		points += 6
	}

	//10 points if the time of purchase is after 2:00pm and before 4:00pm.
	hoursMinutes := date.Hour()*100 + date.Minute()
	if hoursMinutes > 1400 && hoursMinutes < 1560 { //1400 represents 2pm, 1560 represents 4pm
		fmt.Printf("10 points - the time is %d:%d, whitch is between 2pm and 4pm\n", date.Hour(), date.Minute())
		points += 10
	}
	fmt.Printf("Total Points: %d: ", points)

	return points, nil
}

func isAlphaNumeric(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

func roundUp(num float64) int64 {
	return int64(num) + 1
}
