/*
 * Copyright 2026 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://www.apache.org/licenses/LICENSE-2.0
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

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

type noInput struct{}

type userProfile struct {
	Traveler       string   `json:"traveler"`
	City           string   `json:"city"`
	Days           int      `json:"days"`
	Interests      []string `json:"interests"`
	Pace           string   `json:"pace"`
	WakeUpStyle    string   `json:"wake_up_style"`
	FoodPreference string   `json:"food_preference"`
}

type travelPolicy struct {
	BudgetCNY          int      `json:"budget_cny"`
	HotelMaxCNY        int      `json:"hotel_max_cny"`
	Transport          string   `json:"transport"`
	MustAvoid          []string `json:"must_avoid"`
	PreferredTimeSlots []string `json:"preferred_time_slots"`
}

type estimateTripCostInput struct {
	HotelCNY       int `json:"hotel_cny" jsonschema_description:"Hotel cost in CNY"`
	TransportCNY   int `json:"transport_cny" jsonschema_description:"Transport cost in CNY"`
	TicketsCNY     int `json:"tickets_cny" jsonschema_description:"Museum or exhibition ticket cost in CNY"`
	MealsCNY       int `json:"meals_cny" jsonschema_description:"Meals cost in CNY"`
	CoffeeCNY      int `json:"coffee_cny" jsonschema_description:"Coffee and snacks cost in CNY"`
	ContingencyCNY int `json:"contingency_cny" jsonschema_description:"Buffer for unexpected costs in CNY"`
}

type estimateTripCostOutput struct {
	TotalCNY int    `json:"total_cny"`
	Within   bool   `json:"within_budget"`
	Comment  string `json:"comment"`
}

type scoreItineraryInput struct {
	InterestFit int `json:"interest_fit" jsonschema_description:"0-10, how well the plan matches museums, exhibitions and coffee"`
	PaceComfort int `json:"pace_comfort" jsonschema_description:"0-10, how relaxed the schedule is"`
	BudgetFit   int `json:"budget_fit" jsonschema_description:"0-10, how well the plan fits the budget"`
	QueueRisk   int `json:"queue_risk" jsonschema_description:"0-10, higher means more queueing risk"`
}

type scoreItineraryOutput struct {
	Score         int      `json:"score"`
	Decision      string   `json:"decision"`
	TradeOffNotes []string `json:"trade_off_notes"`
}

func travelTools() ([]tool.BaseTool, error) {
	profileTool, err := utils.InferTool("lookup_user_profile", "Return the mock traveler profile for this demo.", lookupUserProfile)
	if err != nil {
		return nil, err
	}

	policyTool, err := utils.InferTool("lookup_travel_policy", "Return the mock budget and travel constraints for this demo.", lookupTravelPolicy)
	if err != nil {
		return nil, err
	}

	costTool, err := utils.InferTool("estimate_trip_cost", "Estimate the trip cost from itemized CNY inputs.", estimateTripCost)
	if err != nil {
		return nil, err
	}

	scoreTool, err := utils.InferTool("score_itinerary", "Score a proposed itinerary from fit, comfort, budget, and queue risk.", scoreItinerary)
	if err != nil {
		return nil, err
	}

	return []tool.BaseTool{profileTool, policyTool, costTool, scoreTool}, nil
}

func lookupUserProfile(_ context.Context, _ *noInput) (*userProfile, error) {
	return &userProfile{
		Traveler:       "Mia",
		City:           "Hangzhou",
		Days:           2,
		Interests:      []string{"exhibitions", "museums", "specialty coffee", "quiet walks"},
		Pace:           "relaxed",
		WakeUpStyle:    "no early mornings",
		FoodPreference: "light meals, one good local dinner",
	}, nil
}

func lookupTravelPolicy(_ context.Context, _ *noInput) (*travelPolicy, error) {
	return &travelPolicy{
		BudgetCNY:   1800,
		HotelMaxCNY: 650,
		Transport:   "metro or taxi for short hops",
		MustAvoid: []string{
			"starting before 10:00",
			"long outdoor exposure in bad weather",
			"more than two high-queue attractions in one day",
		},
		PreferredTimeSlots: []string{"late morning", "early afternoon", "coffee break", "early evening"},
	}, nil
}

func estimateTripCost(_ context.Context, in *estimateTripCostInput) (*estimateTripCostOutput, error) {
	total := in.HotelCNY + in.TransportCNY + in.TicketsCNY + in.MealsCNY + in.CoffeeCNY + in.ContingencyCNY
	return &estimateTripCostOutput{
		TotalCNY: total,
		Within:   total <= 1800,
		Comment:  fmt.Sprintf("estimated total is %d CNY against the 1800 CNY demo budget", total),
	}, nil
}

func scoreItinerary(_ context.Context, in *scoreItineraryInput) (*scoreItineraryOutput, error) {
	score := in.InterestFit*35 + in.PaceComfort*30 + in.BudgetFit*25 - in.QueueRisk*10
	score = score / 9
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	decision := "revise"
	if score >= 80 {
		decision = "recommend"
	} else if score >= 65 {
		decision = "acceptable with backup"
	}

	return &scoreItineraryOutput{
		Score:    score,
		Decision: decision,
		TradeOffNotes: []string{
			"favor fewer stops over maximum coverage",
			"keep a nearby coffee backup for weather or queue changes",
			"reserve ticketed exhibitions when possible",
		},
	}, nil
}
