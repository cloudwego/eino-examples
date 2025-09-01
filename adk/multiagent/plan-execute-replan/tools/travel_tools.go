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

package tools

import (
	"context"
	"fmt"
	"hash/fnv"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// WeatherRequest represents a weather query request
type WeatherRequest struct {
	City string `json:"city" jsonschema:"description=City name to get weather for"`
	Date string `json:"date" jsonschema:"description=Date in YYYY-MM-DD format (optional)"`
}

// WeatherResponse represents weather information
type WeatherResponse struct {
	City        string `json:"city"`
	Temperature int    `json:"temperature"`
	Condition   string `json:"condition"`
	Date        string `json:"date"`
	Error       string `json:"error,omitempty"`
}

// FlightRequest represents a flight search request
type FlightRequest struct {
	From       string `json:"from" jsonschema:"description=Departure city"`
	To         string `json:"to" jsonschema:"description=Destination city"`
	Date       string `json:"date" jsonschema:"description=Departure date in YYYY-MM-DD format"`
	Passengers int    `json:"passengers" jsonschema:"description=Number of passengers"`
}

// FlightResponse represents flight search results
type FlightResponse struct {
	Flights []Flight `json:"flights"`
	Error   string   `json:"error,omitempty"`
}

type Flight struct {
	Airline   string `json:"airline"`
	FlightNo  string `json:"flight_no"`
	Departure string `json:"departure"`
	Arrival   string `json:"arrival"`
	Price     int    `json:"price"`
	Duration  string `json:"duration"`
}

// HotelRequest represents a hotel search request
type HotelRequest struct {
	City     string `json:"city" jsonschema:"description=City to search hotels in"`
	CheckIn  string `json:"check_in" jsonschema:"description=Check-in date in YYYY-MM-DD format"`
	CheckOut string `json:"check_out" jsonschema:"description=Check-out date in YYYY-MM-DD format"`
	Guests   int    `json:"guests" jsonschema:"description=Number of guests"`
}

// HotelResponse represents hotel search results
type HotelResponse struct {
	Hotels []Hotel `json:"hotels"`
	Error  string  `json:"error,omitempty"`
}

type Hotel struct {
	Name      string   `json:"name"`
	Rating    float64  `json:"rating"`
	Price     int      `json:"price"`
	Location  string   `json:"location"`
	Amenities []string `json:"amenities"`
}

// AttractionRequest represents a tourist attraction search request
type AttractionRequest struct {
	City     string `json:"city" jsonschema:"description=City to search attractions in"`
	Category string `json:"category" jsonschema:"description=Category of attractions (museum, park, landmark, historic site, etc.)"`
}

// AttractionResponse represents attraction search results
type AttractionResponse struct {
	Attractions []Attraction `json:"attractions"`
	Error       string       `json:"error,omitempty"`
}

type Attraction struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Rating      float64 `json:"rating"`
	OpenHours   string  `json:"open_hours"`
	TicketPrice int     `json:"ticket_price"`
	Category    string  `json:"category"`
}

// NewWeatherTool Mock weather tool implementation
func NewWeatherTool(ctx context.Context) (tool.BaseTool, error) {
	return utils.InferTool("get_weather", "Get weather information for a specific city and date",
		func(ctx context.Context, req *WeatherRequest) (*WeatherResponse, error) {
			if req.City == "" {
				return &WeatherResponse{Error: "City is required"}, nil
			}

			// Mock weather data
			weathers := map[string]WeatherResponse{
				"Beijing":  {City: "Beijing", Temperature: 15, Condition: "Sunny", Date: req.Date},
				"Shanghai": {City: "Shanghai", Temperature: 20, Condition: "Cloudy", Date: req.Date},
				"Tokyo":    {City: "Tokyo", Temperature: 18, Condition: "Rainy", Date: req.Date},
				"Paris":    {City: "Paris", Temperature: 12, Condition: "Overcast", Date: req.Date},
				"New York": {City: "New York", Temperature: 8, Condition: "Snow", Date: req.Date},
			}

			if weather, exists := weathers[req.City]; exists {
				return &weather, nil
			}

			// Generate consistent weather for unknown cities based on city and date
			conditions := []string{"Sunny", "Cloudy", "Rainy", "Overcast"}
			hashInput := req.City + req.Date
			return &WeatherResponse{
				City:        req.City,
				Temperature: consistentHashing(hashInput+"temp", 5, 35), // 5-35°C
				Condition:   conditions[consistentHashing(hashInput+"cond", 0, len(conditions)-1)],
				Date:        req.Date,
			}, nil
		})
}

// NewFlightSearchTool Mock flight search tool implementation
func NewFlightSearchTool(ctx context.Context) (tool.BaseTool, error) {
	return utils.InferTool("search_flights", "Search for flights between cities",
		func(ctx context.Context, req *FlightRequest) (*FlightResponse, error) {
			if req.From == "" || req.To == "" {
				return &FlightResponse{Error: "From and To cities are required"}, nil
			}

			// Mock flight data
			airlines := []string{"Air China", "China Eastern", "China Southern", "United Airlines", "Delta"}

			flights := make([]Flight, 3)
			hashInput := req.From + req.To + req.Date
			for i := 0; i < 3; i++ {
				flightHash := fmt.Sprintf("%s%d", hashInput, i)
				airlineIdx := consistentHashing(flightHash+"airline", 0, len(airlines)-1)

				// Generate departure and arrival times
				depHour := consistentHashing(flightHash+"dephour", 0, 23)
				depMin := consistentHashing(flightHash+"depmin", 0, 59)
				arrHour := consistentHashing(flightHash+"arrhour", 0, 23)
				arrMin := consistentHashing(flightHash+"arrmin", 0, 59)

				// Calculate duration based on departure and arrival times
				depTotalMin := depHour*60 + depMin
				arrTotalMin := arrHour*60 + arrMin

				// Handle case where arrival is next day (if arrival < departure)
				if arrTotalMin <= depTotalMin {
					arrTotalMin += 24 * 60 // Add 24 hours
				}

				durationMin := arrTotalMin - depTotalMin
				durationHour := durationMin / 60
				durationMinRemainder := durationMin % 60

				flights[i] = Flight{
					Airline:   airlines[airlineIdx],
					FlightNo:  fmt.Sprintf("%s%d", airlines[airlineIdx][:2], consistentHashing(flightHash+"flightno", 1000, 9999)),
					Departure: fmt.Sprintf("%02d:%02d", depHour, depMin),
					Arrival:   fmt.Sprintf("%02d:%02d", arrHour, arrMin),
					Price:     consistentHashing(flightHash+"price", 500, 2500), // $500-2500
					Duration:  fmt.Sprintf("%dh %dm", durationHour, durationMinRemainder),
				}
			}

			return &FlightResponse{Flights: flights}, nil
		})
}

// NewHotelSearchTool Mock hotel search tool implementation
func NewHotelSearchTool(ctx context.Context) (tool.BaseTool, error) {
	return utils.InferTool("search_hotels", "Search for hotels in a city",
		func(ctx context.Context, req *HotelRequest) (*HotelResponse, error) {
			if req.City == "" {
				return &HotelResponse{Error: "City is required"}, nil
			}

			// Mock hotel data
			hotelNames := []string{"Grand Hotel", "City Center Inn", "Luxury Resort", "Budget Lodge", "Business Hotel"}
			amenities := [][]string{
				{"WiFi", "Pool", "Gym", "Spa"},
				{"WiFi", "Breakfast", "Parking"},
				{"WiFi", "Pool", "Restaurant", "Bar", "Concierge"},
				{"WiFi", "Breakfast"},
				{"WiFi", "Business Center", "Meeting Rooms"},
			}

			hotels := make([]Hotel, 4)
			hashInput := req.City + req.CheckIn + req.CheckOut
			for i := 0; i < 4; i++ {
				hotelHash := fmt.Sprintf("%s%d", hashInput, i)
				hotels[i] = Hotel{
					Name:      fmt.Sprintf("%s %s", req.City, hotelNames[consistentHashing(hotelHash+"name", 0, len(hotelNames)-1)]),
					Rating:    float64(consistentHashing(hotelHash+"rating", 20, 50)) / 10.0, // 2.0-5.0
					Price:     consistentHashing(hotelHash+"price", 50, 350),                 // $50-350 per night
					Location:  fmt.Sprintf("%s Downtown", req.City),
					Amenities: amenities[consistentHashing(hotelHash+"amenities", 0, len(amenities)-1)],
				}
			}

			return &HotelResponse{Hotels: hotels}, nil
		})
}

// NewAttractionSearchTool Mock attraction search tool implementation
func NewAttractionSearchTool(ctx context.Context) (tool.BaseTool, error) {
	return utils.InferTool("search_attractions", "Search for tourist attractions in a city",
		func(ctx context.Context, req *AttractionRequest) (*AttractionResponse, error) {
			if req.City == "" {
				return &AttractionResponse{Error: "City is required"}, nil
			}

			// Mock attraction data based on city
			attractionsByCity := map[string][]Attraction{
				"Beijing": {
					{Name: "Forbidden City", Description: "Ancient imperial palace", Rating: 4.8, OpenHours: "8:30-17:00", TicketPrice: 60, Category: "historic site"},
					{Name: "Great Wall", Description: "Historic fortification", Rating: 4.9, OpenHours: "6:00-18:00", TicketPrice: 45, Category: "landmark"},
					{Name: "Temple of Heaven", Description: "Imperial sacrificial altar", Rating: 4.6, OpenHours: "6:00-22:00", TicketPrice: 35, Category: "park"},
				},
				"Paris": {
					{Name: "Eiffel Tower", Description: "Iconic iron lattice tower", Rating: 4.7, OpenHours: "9:30-23:45", TicketPrice: 25, Category: "landmark"},
					{Name: "Louvre Museum", Description: "World's largest art museum", Rating: 4.8, OpenHours: "9:00-18:00", TicketPrice: 17, Category: "museum"},
					{Name: "Notre-Dame Cathedral", Description: "Medieval Catholic cathedral", Rating: 4.5, OpenHours: "8:00-18:45", TicketPrice: 0, Category: "landmark"},
				},
				"Tokyo": {
					{Name: "Senso-ji Temple", Description: "Ancient Buddhist temple", Rating: 4.4, OpenHours: "6:00-17:00", TicketPrice: 0, Category: "landmark"},
					{Name: "Tokyo National Museum", Description: "Largest collection of cultural artifacts", Rating: 4.3, OpenHours: "9:30-17:00", TicketPrice: 1000, Category: "museum"},
					{Name: "Ueno Park", Description: "Large public park with museums", Rating: 4.2, OpenHours: "5:00-23:00", TicketPrice: 0, Category: "park"},
				},
			}

			if attractions, exists := attractionsByCity[req.City]; exists {
				// Filter by category if specified
				if req.Category != "" {
					var filtered []Attraction
					for _, attraction := range attractions {
						if attraction.Category == req.Category {
							filtered = append(filtered, attraction)
						}
					}
					return &AttractionResponse{Attractions: filtered}, nil
				}
				return &AttractionResponse{Attractions: attractions}, nil
			}

			// Generate random attractions for unknown cities
			attractionNames := []string{"Central Museum", "City Park", "Historic Square", "Art Gallery", "Cultural Center"}
			categories := []string{"museum", "park", "landmark", "historic site", "cultural"}

			attractions := make([]Attraction, 3)
			hashInput := req.City + req.Category
			for i := 0; i < 3; i++ {
				attractionHash := fmt.Sprintf("%s%d", hashInput, i)
				attractions[i] = Attraction{
					Name:        fmt.Sprintf("%s %s", req.City, attractionNames[consistentHashing(attractionHash+"name", 0, len(attractionNames)-1)]),
					Description: "Popular tourist attraction",
					Rating:      float64(consistentHashing(attractionHash+"rating", 30, 50)) / 10.0, // 3.0-5.0
					OpenHours:   "9:00-17:00",
					TicketPrice: consistentHashing(attractionHash+"price", 0, 50),
					Category:    categories[consistentHashing(attractionHash+"category", 0, len(categories)-1)],
				}
			}

			return &AttractionResponse{Attractions: attractions}, nil
		})
}

// GetAllTravelTools returns all travel-related tools
func GetAllTravelTools(ctx context.Context) ([]tool.BaseTool, error) {
	weatherTool, err := NewWeatherTool(ctx)
	if err != nil {
		return nil, err
	}

	flightTool, err := NewFlightSearchTool(ctx)
	if err != nil {
		return nil, err
	}

	hotelTool, err := NewHotelSearchTool(ctx)
	if err != nil {
		return nil, err
	}

	attractionTool, err := NewAttractionSearchTool(ctx)
	if err != nil {
		return nil, err
	}

	askForClarificationTool := NewAskForClarificationTool()

	return []tool.BaseTool{weatherTool, flightTool, hotelTool, attractionTool, askForClarificationTool}, nil
}

// consistentHashing implements consistent hashing using Go standard library hash/fnv
func consistentHashing(s string, min, max int) int {
	// Use FNV-1a hash algorithm from Go standard library
	h := fnv.New32a()
	h.Write([]byte(s))
	hash := h.Sum32()

	// Map to range [min, max]
	return min + int(hash)%(max-min+1)
}
