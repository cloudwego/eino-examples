/*
 * Copyright 2025 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"context"
	"fmt"
	"hash/fnv"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"

	tool2 "github.com/cloudwego/eino-examples/adk/common/tool"
)

type WeatherRequest struct {
	City string `json:"city" jsonschema_description:"City name to get weather for"`
	Date string `json:"date" jsonschema_description:"Date in YYYY-MM-DD format"`
}

type WeatherResponse struct {
	City        string `json:"city"`
	Temperature int    `json:"temperature"`
	Condition   string `json:"condition"`
	Date        string `json:"date"`
}

type FlightBookingRequest struct {
	From          string `json:"from" jsonschema_description:"Departure city"`
	To            string `json:"to" jsonschema_description:"Destination city"`
	Date          string `json:"date" jsonschema_description:"Departure date in YYYY-MM-DD format"`
	Passengers    int    `json:"passengers" jsonschema_description:"Number of passengers"`
	PreferredTime string `json:"preferred_time" jsonschema_description:"Preferred departure time (morning/afternoon/evening)"`
}

type FlightBookingResponse struct {
	BookingID     string `json:"booking_id"`
	Airline       string `json:"airline"`
	FlightNo      string `json:"flight_no"`
	From          string `json:"from"`
	To            string `json:"to"`
	Date          string `json:"date"`
	DepartureTime string `json:"departure_time"`
	ArrivalTime   string `json:"arrival_time"`
	Price         int    `json:"price"`
	Status        string `json:"status"`
}

type HotelBookingRequest struct {
	City     string `json:"city" jsonschema_description:"City to book hotel in"`
	CheckIn  string `json:"check_in" jsonschema_description:"Check-in date in YYYY-MM-DD format"`
	CheckOut string `json:"check_out" jsonschema_description:"Check-out date in YYYY-MM-DD format"`
	Guests   int    `json:"guests" jsonschema_description:"Number of guests"`
	RoomType string `json:"room_type" jsonschema_description:"Room type preference (standard/deluxe/suite)"`
}

type HotelBookingResponse struct {
	BookingID     string   `json:"booking_id"`
	HotelName     string   `json:"hotel_name"`
	City          string   `json:"city"`
	CheckIn       string   `json:"check_in"`
	CheckOut      string   `json:"check_out"`
	RoomType      string   `json:"room_type"`
	PricePerNight int      `json:"price_per_night"`
	TotalPrice    int      `json:"total_price"`
	Amenities     []string `json:"amenities"`
	Status        string   `json:"status"`
}

type AttractionRequest struct {
	City     string `json:"city" jsonschema_description:"City to search attractions in"`
	Category string `json:"category" jsonschema_description:"Category of attractions (museum, park, landmark, etc.)"`
}

type AttractionResponse struct {
	Attractions []Attraction `json:"attractions"`
}

type Attraction struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Rating      float64 `json:"rating"`
	OpenHours   string  `json:"open_hours"`
	TicketPrice int     `json:"ticket_price"`
	Category    string  `json:"category"`
}

func NewWeatherTool(ctx context.Context) (tool.BaseTool, error) {
	return utils.InferTool("get_weather", "Get weather information for a specific city and date",
		func(ctx context.Context, req *WeatherRequest) (*WeatherResponse, error) {
			weathers := map[string]WeatherResponse{
				"Tokyo":    {City: "Tokyo", Temperature: 22, Condition: "Partly Cloudy", Date: req.Date},
				"Beijing":  {City: "Beijing", Temperature: 18, Condition: "Sunny", Date: req.Date},
				"Paris":    {City: "Paris", Temperature: 15, Condition: "Rainy", Date: req.Date},
				"New York": {City: "New York", Temperature: 12, Condition: "Cloudy", Date: req.Date},
			}

			if weather, exists := weathers[req.City]; exists {
				return &weather, nil
			}

			conditions := []string{"Sunny", "Cloudy", "Rainy", "Partly Cloudy"}
			hashInput := req.City + req.Date
			return &WeatherResponse{
				City:        req.City,
				Temperature: consistentHashing(hashInput+"temp", 10, 30),
				Condition:   conditions[consistentHashing(hashInput+"cond", 0, len(conditions)-1)],
				Date:        req.Date,
			}, nil
		})
}

func NewFlightBookingTool(ctx context.Context) (tool.BaseTool, error) {
	baseTool, err := utils.InferTool("book_flight", "Book a flight between cities. This requires user review before confirmation.",
		func(ctx context.Context, req *FlightBookingRequest) (*FlightBookingResponse, error) {
			airlines := []string{"Japan Airlines", "ANA", "United Airlines", "Delta", "Air China"}
			hashInput := req.From + req.To + req.Date

			airlineIdx := consistentHashing(hashInput+"airline", 0, len(airlines)-1)
			depHour := consistentHashing(hashInput+"dephour", 6, 20)
			depMin := consistentHashing(hashInput+"depmin", 0, 59)
			duration := consistentHashing(hashInput+"duration", 2, 14)

			arrHour := (depHour + duration) % 24
			arrMin := depMin

			return &FlightBookingResponse{
				BookingID:     fmt.Sprintf("FL-%s-%d", req.Date, consistentHashing(hashInput+"id", 10000, 99999)),
				Airline:       airlines[airlineIdx],
				FlightNo:      fmt.Sprintf("%s%d", airlines[airlineIdx][:2], consistentHashing(hashInput+"flightno", 100, 999)),
				From:          req.From,
				To:            req.To,
				Date:          req.Date,
				DepartureTime: fmt.Sprintf("%02d:%02d", depHour, depMin),
				ArrivalTime:   fmt.Sprintf("%02d:%02d", arrHour, arrMin),
				Price:         consistentHashing(hashInput+"price", 300, 1500) * req.Passengers,
				Status:        "confirmed",
			}, nil
		})
	if err != nil {
		return nil, err
	}

	return &tool2.InvokableReviewEditTool{InvokableTool: baseTool}, nil
}

func NewHotelBookingTool(ctx context.Context) (tool.BaseTool, error) {
	baseTool, err := utils.InferTool("book_hotel", "Book a hotel in a city. This requires user review before confirmation.",
		func(ctx context.Context, req *HotelBookingRequest) (*HotelBookingResponse, error) {
			hotelNames := []string{"Grand Hyatt", "Marriott", "Hilton", "Sheraton", "Ritz-Carlton"}
			amenitiesList := [][]string{
				{"WiFi", "Pool", "Gym", "Spa", "Restaurant"},
				{"WiFi", "Breakfast", "Parking", "Business Center"},
				{"WiFi", "Pool", "Restaurant", "Bar", "Concierge"},
			}

			hashInput := req.City + req.CheckIn + req.CheckOut

			hotelIdx := consistentHashing(hashInput+"hotel", 0, len(hotelNames)-1)
			amenitiesIdx := consistentHashing(hashInput+"amenities", 0, len(amenitiesList)-1)

			pricePerNight := consistentHashing(hashInput+"price", 100, 400)
			if req.RoomType == "deluxe" {
				pricePerNight = int(float64(pricePerNight) * 1.5)
			} else if req.RoomType == "suite" {
				pricePerNight = pricePerNight * 2
			}

			nights := 3

			return &HotelBookingResponse{
				BookingID:     fmt.Sprintf("HT-%s-%d", req.CheckIn, consistentHashing(hashInput+"id", 10000, 99999)),
				HotelName:     fmt.Sprintf("%s %s", req.City, hotelNames[hotelIdx]),
				City:          req.City,
				CheckIn:       req.CheckIn,
				CheckOut:      req.CheckOut,
				RoomType:      req.RoomType,
				PricePerNight: pricePerNight,
				TotalPrice:    pricePerNight * nights,
				Amenities:     amenitiesList[amenitiesIdx],
				Status:        "confirmed",
			}, nil
		})
	if err != nil {
		return nil, err
	}

	return &tool2.InvokableReviewEditTool{InvokableTool: baseTool}, nil
}

func NewAttractionSearchTool(ctx context.Context) (tool.BaseTool, error) {
	return utils.InferTool("search_attractions", "Search for tourist attractions in a city",
		func(ctx context.Context, req *AttractionRequest) (*AttractionResponse, error) {
			attractionsByCity := map[string][]Attraction{
				"Tokyo": {
					{Name: "Senso-ji Temple", Description: "Ancient Buddhist temple in Asakusa", Rating: 4.6, OpenHours: "6:00-17:00", TicketPrice: 0, Category: "landmark"},
					{Name: "Tokyo Skytree", Description: "Tallest tower in Japan with observation decks", Rating: 4.5, OpenHours: "10:00-21:00", TicketPrice: 2100, Category: "landmark"},
					{Name: "Meiji Shrine", Description: "Shinto shrine dedicated to Emperor Meiji", Rating: 4.7, OpenHours: "5:00-18:00", TicketPrice: 0, Category: "landmark"},
					{Name: "Ueno Park", Description: "Large public park with museums and zoo", Rating: 4.4, OpenHours: "5:00-23:00", TicketPrice: 0, Category: "park"},
				},
				"Beijing": {
					{Name: "Forbidden City", Description: "Ancient imperial palace complex", Rating: 4.8, OpenHours: "8:30-17:00", TicketPrice: 60, Category: "historic site"},
					{Name: "Great Wall", Description: "Historic fortification stretching thousands of miles", Rating: 4.9, OpenHours: "6:00-18:00", TicketPrice: 45, Category: "landmark"},
					{Name: "Temple of Heaven", Description: "Imperial sacrificial altar", Rating: 4.6, OpenHours: "6:00-22:00", TicketPrice: 35, Category: "park"},
				},
			}

			if attractions, exists := attractionsByCity[req.City]; exists {
				if req.Category != "" {
					var filtered []Attraction
					for _, a := range attractions {
						if a.Category == req.Category {
							filtered = append(filtered, a)
						}
					}
					return &AttractionResponse{Attractions: filtered}, nil
				}
				return &AttractionResponse{Attractions: attractions}, nil
			}

			return &AttractionResponse{
				Attractions: []Attraction{
					{Name: fmt.Sprintf("%s Central Park", req.City), Description: "Popular city park", Rating: 4.3, OpenHours: "6:00-22:00", TicketPrice: 0, Category: "park"},
					{Name: fmt.Sprintf("%s National Museum", req.City), Description: "Major museum with local history", Rating: 4.5, OpenHours: "9:00-17:00", TicketPrice: 15, Category: "museum"},
				},
			}, nil
		})
}

func GetAllTravelTools(ctx context.Context) ([]tool.BaseTool, error) {
	weatherTool, err := NewWeatherTool(ctx)
	if err != nil {
		return nil, err
	}

	flightTool, err := NewFlightBookingTool(ctx)
	if err != nil {
		return nil, err
	}

	hotelTool, err := NewHotelBookingTool(ctx)
	if err != nil {
		return nil, err
	}

	attractionTool, err := NewAttractionSearchTool(ctx)
	if err != nil {
		return nil, err
	}

	return []tool.BaseTool{weatherTool, flightTool, hotelTool, attractionTool}, nil
}

func consistentHashing(s string, min, max int) int {
	h := fnv.New32a()
	h.Write([]byte(s))
	hash := h.Sum32()
	return min + int(hash)%(max-min+1)
}
