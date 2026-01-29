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

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

type GetWeatherInput struct {
	Location string `json:"location" jsonschema:"description=The city and state, e.g. San Francisco, CA"`
	Unit     string `json:"unit,omitempty" jsonschema:"enum=celsius,enum=fahrenheit,description=The unit of temperature"`
}

type GetWeatherOutput struct {
	Temperature float64 `json:"temperature"`
	Unit        string  `json:"unit"`
	Condition   string  `json:"condition"`
}

type GetForecastInput struct {
	Location string `json:"location" jsonschema:"description=The city and state"`
	Days     int    `json:"days" jsonschema:"description=Number of days to forecast (1-10)"`
}

type GetForecastOutput struct {
	Forecasts []DayForecast `json:"forecasts"`
}

type DayForecast struct {
	Day         string  `json:"day"`
	Temperature float64 `json:"temperature"`
	Condition   string  `json:"condition"`
}

type GetStockPriceInput struct {
	Ticker         string `json:"ticker" jsonschema:"description=Stock ticker symbol (e.g., AAPL, GOOGL)"`
	IncludeHistory bool   `json:"include_history,omitempty" jsonschema:"description=Include historical data"`
}

type GetStockPriceOutput struct {
	Ticker string  `json:"ticker"`
	Price  float64 `json:"price"`
	Change float64 `json:"change"`
}

type ConvertCurrencyInput struct {
	Amount       float64 `json:"amount" jsonschema:"description=Amount to convert"`
	FromCurrency string  `json:"from_currency" jsonschema:"description=Source currency code (e.g., USD)"`
	ToCurrency   string  `json:"to_currency" jsonschema:"description=Target currency code (e.g., EUR)"`
}

type ConvertCurrencyOutput struct {
	OriginalAmount  float64 `json:"original_amount"`
	ConvertedAmount float64 `json:"converted_amount"`
	ExchangeRate    float64 `json:"exchange_rate"`
}

func createWeatherTools() []tool.BaseTool {
	getWeather, _ := utils.InferTool(
		"get_weather",
		"Get the current weather in a given location",
		func(ctx context.Context, input *GetWeatherInput) (*GetWeatherOutput, error) {
			return &GetWeatherOutput{
				Temperature: 22.5,
				Unit:        input.Unit,
				Condition:   "Sunny",
			}, nil
		},
	)

	getForecast, _ := utils.InferTool(
		"get_forecast",
		"Get the weather forecast for multiple days ahead",
		func(ctx context.Context, input *GetForecastInput) (*GetForecastOutput, error) {
			forecasts := make([]DayForecast, input.Days)
			for i := 0; i < input.Days; i++ {
				forecasts[i] = DayForecast{
					Day:         fmt.Sprintf("Day %d", i+1),
					Temperature: 20.0 + float64(i),
					Condition:   "Partly Cloudy",
				}
			}
			return &GetForecastOutput{Forecasts: forecasts}, nil
		},
	)

	return []tool.BaseTool{getWeather, getForecast}
}

func createFinanceTools() []tool.BaseTool {
	getStockPrice, _ := utils.InferTool(
		"get_stock_price",
		"Get the current stock price and market data for a given ticker symbol",
		func(ctx context.Context, input *GetStockPriceInput) (*GetStockPriceOutput, error) {
			return &GetStockPriceOutput{
				Ticker: input.Ticker,
				Price:  150.25,
				Change: 2.5,
			}, nil
		},
	)

	convertCurrency, _ := utils.InferTool(
		"convert_currency",
		"Convert an amount from one currency to another using current exchange rates",
		func(ctx context.Context, input *ConvertCurrencyInput) (*ConvertCurrencyOutput, error) {
			rate := 0.85
			return &ConvertCurrencyOutput{
				OriginalAmount:  input.Amount,
				ConvertedAmount: input.Amount * rate,
				ExchangeRate:    rate,
			}, nil
		},
	)

	return []tool.BaseTool{getStockPrice, convertCurrency}
}
