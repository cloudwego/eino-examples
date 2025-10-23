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
	"sort"
	"time"
)

type ActivityType string

const (
	ActivityTypeAttraction  ActivityType = "attraction"
	ActivityTypePerformance ActivityType = "performance"
	ActivityTypeRestaurant  ActivityType = "restaurant"
	ActivityTypeOther       ActivityType = "other"
)

// Activity 主题乐园中的一个项目，可以是游乐设施、表演或餐厅.
type Activity struct {
	Name               string       `json:"name"`
	Desc               string       `json:"desc"`
	Type               ActivityType `json:"type"`
	Location           string       `json:"location" jsonschema_description:"项目所属的区域"`
	MinHeight          int          `json:"min_height,omitempty" jsonschema_description:"参加游乐设施需要的最小身高，单位是厘米。如果为空，则没有身高要求"`
	Duration           int          `json:"duration,omitempty" jsonschema_description:"一个项目参加一次需要的时间，注意不包括排队的时间。如果为空，则缺少具体的时间信息，可以默认为 10 分钟"`
	TimeTable          []string     `json:"time_table,omitempty" jsonschema_description:"一个演出的时间表。如果为空，则使用 OpenTime 和 CloseTime 来表示这个项目的运营时间范围"`
	OpenTime           string       `json:"open_time,omitempty" jsonschema_description:"一个项目开始运营的时间"`
	CloseTime          string       `json:"close_time,omitempty" jsonschema_description:"一个项目结束运营的时间"`
	RequireBooking     bool         `json:"require_booking,omitempty" jsonschema_description:"一个餐厅是否需要提前预约"`
	HasPriorityAccess  bool         `json:"has_priority_access,omitempty" jsonschema_description:"一个项目是否有高速票服务"`
	PriorityAccessCost int          `json:"priority_access_cost,omitempty" jsonschema_description:"一个项目如果有高速票服务，则一个人的高速票需要花多少钱"`
	QueueTime          float64      `json:"queue_time,omitempty" jsonschema_description:"一个项目常规需要的排队时间，单位是分钟。如果为空，则这个项目一般不需要排队"`
}

// LocationAdjacency 主题乐园的一个区域到相邻区域的步行时间.
type LocationAdjacency struct {
	FromLocationName                string                   `json:"from_location_name" jsonschema_description:"从哪个区域开始计算距离"`
	DestinationLocationWalkingTimes []DestinationWalkingTime `json:"destination_location_walking_times,omitempty" jsonschema_description:"相邻区域列表，包含走路的分钟数"`
}

type DestinationWalkingTime struct {
	DestinationName string  `json:"destination_name" jsonschema_description:"目标区域名称"`
	WalkTime        float64 `json:"walk_time" jsonschema_description:"步行到目标区域所需要的分钟数"`
}

// AttractionQueueTime 主题乐园的一个游乐项目的排队时间.
type AttractionQueueTime struct {
	Name      string  `json:"name" jsonschema_description:"游乐项目的名称"`
	QueueTime float64 `json:"queue_time" jsonschema_description:"游乐项目的排队时间"`
}

type ListPerformanceRequest struct {
	Name     string `json:"name,omitempty" jsonschema_description:"演出的名称，如果不需要查询具体的某个演出，此处传空"`
	Location string `json:"location,omitempty" jsonschema_description:"演出所在的区域，如果不需要查询具体某个区域的演出，此处传空"`
}
type ListPerformanceResponse struct {
	Performances []Activity `json:"performances" jsonschema_description:"符合查询条件的所有演出的信息"`
}

type ListAttractionRequest struct {
	Name     string `json:"name,omitempty" jsonschema_description:"游乐项目的名称，如果不需要查询具体的某个游乐项目，此处传空"`
	Location string `json:"location,omitempty"  jsonschema_description:"游乐项目所在的区域，如果不需要查询具体某个区域的游乐项目，此处传空"`
}
type ListAttractionResponse struct {
	Attractions []Activity `json:"attractions" jsonschema_description:"符合查询条件的所有游乐项目的信息"`
}

type ListRestaurantRequest struct {
	Name     string `json:"name,omitempty" jsonschema_description:"餐厅的名称，如果不需要查询具体的某个餐厅，此处传空"`
	Location string `json:"location,omitempty"  jsonschema_description:"餐厅所在的区域，如果不需要查询具体某个区域的餐厅，此处传空"`
}
type ListRestaurantResponse struct {
	Restaurants []Activity `json:"restaurants" jsonschema_description:"符合查询条件的所有餐厅的信息"`
}

type ListAttractionQueueTimeRequest struct {
	Name     string `json:"name,omitempty" jsonschema_description:"游乐项目的名称，如果不需要查询具体的某个游乐项目的排队时间，此处传空"`
	Location string `json:"location,omitempty"  jsonschema_description:"游乐项目所在的区域，如果不需要查询具体某个区域的游乐项目的排队时间，此处传空"`
}
type ListAttractionQueueTimeResponse struct {
	QueueTime []AttractionQueueTime `json:"queue_time" jsonschema_description:"符合查询条件的所有游乐项目的排队时间"`
}

type ListAdjacentLocationRequest struct{}
type ListAdjacentLocationResponse struct {
	AdjacencyList []LocationAdjacency `json:"adjacency_list" jsonschema_description:"所有区域的邻接区域"`
}

type GetParkHourRequest struct{}
type GetParkHourResponse struct {
	OpenHour  string `json:"open_hour"`
	CloseHour string `json:"close_hour"`
}

type GetParkTicketPriceRequest struct{}
type GetParkTicketPriceResponse struct {
	Price string `json:"price" jsonschema_description:"乐园门票价格信息"`
}

type ListLocationsRequest struct{}
type ListLocationsResponse struct {
	Locations []string `json:"locations"`
}

func ListLocations(_ context.Context, _ *ListLocationsRequest) (out *ListLocationsResponse, err error) {
	return &ListLocationsResponse{
		Locations: []string{
			"幻想世界",
			"未来世界",
			"冒险岛",
			"宝贝港湾",
			"入口大街",
			"奇幻园林",
			"热情小动物城市",
			"玩具的故事",
		},
	}, nil
}

type QueryEntranceRequest struct{}
type QueryEntranceResponse struct {
	EntranceLocation string `json:"entrance_location" jsonschema_description:"园区入口区域名称"`
}

func QueryEntrance(_ context.Context, _ *QueryEntranceRequest) (out *QueryEntranceResponse, err error) {
	return &QueryEntranceResponse{EntranceLocation: "入口大街"}, nil
}

// GetAdjacentLocation 获取相邻的地点，可获取全部地点到其邻接地点的步行分钟数，也可获取单个地点到其邻接地点的步行分钟数.
func GetAdjacentLocation(_ context.Context, _ *ListAdjacentLocationRequest) (out *ListAdjacentLocationResponse, err error) {
	adjacencyMap := make(map[string][]DestinationWalkingTime)
	for k, v := range locationAdjacencyMap {
		if _, ok := adjacencyMap[k]; !ok {
			adjacencyMap[k] = make([]DestinationWalkingTime, 0)
		}

		for dest, walkingTime := range v {
			adjacencyMap[k] = append(adjacencyMap[k], DestinationWalkingTime{
				DestinationName: dest,
				WalkTime:        walkingTime,
			})
		}
	}

	adjacencyList := make([]LocationAdjacency, 0)
	for k, v := range adjacencyMap {
		adjacencyList = append(adjacencyList, LocationAdjacency{
			FromLocationName:                k,
			DestinationLocationWalkingTimes: v,
		})
	}

	return &ListAdjacentLocationResponse{
		AdjacencyList: adjacencyList,
	}, nil
}

func GetParkTicketPrice(_ context.Context, _ *GetParkTicketPriceRequest) (out *GetParkTicketPriceResponse, err error) {
	return &GetParkTicketPriceResponse{Price: "成人票 400，儿童票 300"}, nil
}

// GetParkHour 获取乐园的营业时间.
func GetParkHour(_ context.Context, _ *GetParkHourRequest) (out *GetParkHourResponse, err error) {
	return &GetParkHourResponse{
		OpenHour:  "09:00",
		CloseHour: "21:30",
	}, nil
}

// GetQueueTime 获取游乐设施排队时间.
func GetQueueTime(_ context.Context, in *ListAttractionQueueTimeRequest) (out *ListAttractionQueueTimeResponse, err error) {
	if len(in.Name) > 0 {
		for _, a := range attractions {
			if a.Name == in.Name {
				return &ListAttractionQueueTimeResponse{
					QueueTime: []AttractionQueueTime{
						{
							Name:      a.Name,
							QueueTime: a.QueueTime,
						},
					},
				}, nil
			}
		}
	}
	if len(in.Location) > 0 {
		queueTimes := make([]AttractionQueueTime, 0)
		for _, a := range attractions {
			if a.Location == in.Location {
				queueTimes = append(queueTimes, AttractionQueueTime{
					Name:      a.Name,
					QueueTime: a.QueueTime,
				})
				return &ListAttractionQueueTimeResponse{
					QueueTime: queueTimes,
				}, nil
			}
		}
	}

	queueTimes := make([]AttractionQueueTime, 0)
	for _, a := range attractions {
		queueTimes = append(queueTimes, AttractionQueueTime{
			Name:      a.Name,
			QueueTime: a.QueueTime,
		})
	}

	return &ListAttractionQueueTimeResponse{
		QueueTime: queueTimes,
	}, nil
}

// GetAttractionInfo 获取游乐设施信息.
func GetAttractionInfo(_ context.Context, in *ListAttractionRequest) (out *ListAttractionResponse, err error) {
	if len(in.Name) > 0 && in.Name != "all" {
		for _, a := range attractions {
			if a.Name == in.Name {
				return &ListAttractionResponse{
					Attractions: []Activity{
						a,
					},
				}, nil
			}
		}
	}

	if len(in.Location) > 0 {
		locationAttractions := make([]Activity, 0)
		for _, a := range attractions {
			if a.Location == in.Location {
				locationAttractions = append(locationAttractions, a)
				return &ListAttractionResponse{
					Attractions: locationAttractions,
				}, nil
			}
		}
	}

	return &ListAttractionResponse{
		Attractions: attractions,
	}, nil
}

// GetPerformanceInfo 获取演出信息.
func GetPerformanceInfo(_ context.Context, in *ListPerformanceRequest) (out *ListPerformanceResponse, err error) {
	if len(in.Name) > 0 && in.Name != "all" {
		for _, a := range performances {
			if a.Name == in.Name {
				return &ListPerformanceResponse{
					Performances: []Activity{
						a,
					},
				}, nil
			}
		}
	}
	if len(in.Location) > 0 {
		locationPerformances := make([]Activity, 0)
		for _, a := range performances {
			if a.Location == in.Location {
				locationPerformances = append(locationPerformances, a)
				return &ListPerformanceResponse{
					Performances: locationPerformances,
				}, nil
			}
		}
	}
	return &ListPerformanceResponse{
		Performances: performances,
	}, nil
}

// GetRestaurantInfo 获取餐厅信息.
func GetRestaurantInfo(_ context.Context, in *ListRestaurantRequest) (out *ListRestaurantResponse, err error) {
	if len(in.Name) > 0 && in.Name != "all" {
		for _, a := range restaurants {
			if a.Name == in.Name {
				return &ListRestaurantResponse{
					Restaurants: []Activity{
						a,
					},
				}, nil
			}
		}
	}

	if len(in.Location) > 0 {
		locationRestaurants := make([]Activity, 0)
		for _, a := range restaurants {
			if a.Location == in.Location {
				locationRestaurants = append(locationRestaurants, a)
				return &ListRestaurantResponse{
					Restaurants: locationRestaurants,
				}, nil
			}
		}
	}

	return &ListRestaurantResponse{
		Restaurants: restaurants,
	}, nil
}

type OnePerformanceStartTime struct {
	PerformanceName string `json:"performance_name"`
	StartTime       string `json:"start_time" jsonschema_description:"选中的演出开始时间，格式如15:30"`
}
type ValidatePerformanceTimeTableRequest struct {
	PerformancesStartTime []OnePerformanceStartTime `json:"performances_start_time" jsonschema_description:"用户选择的演出名称和开始时间"`
}
type PerformanceStartTimeValidateResult struct {
	PerformanceName string `json:"performance_name"`
	StartTime       string `json:"start_time"`
	IsValid         bool   `json:"is_valid"`
	ErrMessage      string `json:"err_message"`
}
type ValidatePerformanceTimeTableResponse struct {
	PerformancesValidateResult []PerformanceStartTimeValidateResult `json:"performances_validate_result" jsonschema_description:"验证结果，只包含有问题的表演"`
}

func ValidatePerformanceTimeTable(_ context.Context, in *ValidatePerformanceTimeTableRequest) (out *ValidatePerformanceTimeTableResponse, err error) {
	results := make([]PerformanceStartTimeValidateResult, 0, len(in.PerformancesStartTime))

	for _, performance := range in.PerformancesStartTime {
		var (
			performanceFound bool
			timeTable        []string
		)
		for _, p := range getPerformances() {
			if p.Name == performance.PerformanceName {
				for _, t := range p.TimeTable {
					if t == performance.StartTime {
						break
					}
				}

				performanceFound = true
				timeTable = p.TimeTable
				break
			}
		}

		if !performanceFound {
			results = append(results, PerformanceStartTimeValidateResult{
				PerformanceName: performance.PerformanceName,
				StartTime:       performance.StartTime,
				IsValid:         false,
				ErrMessage:      fmt.Sprintf("传入的表演名称为 %s，未找到对应的表演", performance.PerformanceName),
			})
		} else if !contains(timeTable, performance.StartTime) {
			results = append(results, PerformanceStartTimeValidateResult{
				PerformanceName: performance.PerformanceName,
				StartTime:       performance.StartTime,
				IsValid:         false,
				ErrMessage:      fmt.Sprintf("传入的表演名称为 %s，传入的开始时间为 %s，但该表演实际的时间表为 %v", performance.PerformanceName, performance.StartTime, timeTable),
			})
		}
	}

	return &ValidatePerformanceTimeTableResponse{
		PerformancesValidateResult: results,
	}, nil
}

// contains is a helper function to check if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

type ArrangePerformancesRequest struct {
	ChosenPerformances []string `json:"chosen_performances" jsonschema_description:"用户选择的演出名称列表"`
}
type ArrangePerformancesResponse struct {
	ArrangedPerformances    []PerformanceTime `json:"arranged_performances" jsonschema_description:"根据用户选择的演出，以及实际的时间表和演出时长，计算出的看演出的时间规划。包含了每场演出的排队、占座时间。"`
	UnsatisfiedPerformances []string          `json:"unsatisfied_performances" jsonschema_description:"由于跟其他演出的时间冲突，无法完成时间安排的演出名称"`
}
type PerformanceTime struct {
	PerformanceName string `json:"performance_name" jsonschema_description:"演出名称"`
	StartTime       string `json:"start_time" jsonschema_description:"演出开始时间"`
	EndTime         string `json:"end_time" jsonschema_description:"演出结束时间"`
}

func getPerformances() []Activity {
	return performances
}

func ArrangePerformances(_ context.Context, in *ArrangePerformancesRequest) (out *ArrangePerformancesResponse, err error) {
	performanceInfos := make(map[string]Activity)
	for _, p := range getPerformances() {
		performanceInfos[p.Name] = p
	}

	chosenPerformanceInfos := make(map[string]Activity, len(in.ChosenPerformances))

	for _, chosen := range in.ChosenPerformances {
		if _, ok := performanceInfos[chosen]; !ok {
			return nil, fmt.Errorf("传入的演出名称为 %s，未找到对应的表演", chosen)
		}
		chosenPerformanceInfos[chosen] = performanceInfos[chosen]
	}

	arranged := make(map[string]PerformanceTime)
	unsatisfied := make(map[string]bool)

	// Define a helper function to parse time and calculate end time
	parseTimeAndEnd := func(startTimeStr string, duration int) (time.Time, time.Time, error) {
		startTime, err := time.Parse("15:04", startTimeStr)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
		endTime := startTime.Add(time.Duration(duration) * time.Minute)
		return startTime, endTime, nil
	}

	// Define a helper function to check if there is a time conflict
	hasConflict := func(startTime, endTime time.Time) bool {
		twentyMinutes := 20 * time.Minute
		for _, perf := range arranged {
			existingStart, err := time.Parse("15:04", perf.StartTime)
			if err != nil {
				continue
			}
			existingEnd, err := time.Parse("15:04", perf.EndTime)
			if err != nil {
				continue
			}
			// 检查时间冲突，包括间隔小于 20 分钟的情况
			if (startTime.Before(existingEnd.Add(twentyMinutes)) && endTime.After(existingStart.Add(-twentyMinutes))) ||
				startTime.Equal(existingEnd.Add(twentyMinutes)) ||
				endTime.Equal(existingStart.Add(-twentyMinutes)) {
				return true
			}
		}
		return false
	}

	// Sort chosen performances by the number of available time slots in ascending order
	sortedChosenPerformances := make([]string, 0, len(chosenPerformanceInfos))
	for name := range chosenPerformanceInfos {
		sortedChosenPerformances = append(sortedChosenPerformances, name)
	}
	sort.Slice(sortedChosenPerformances, func(i, j int) bool {
		return len(chosenPerformanceInfos[sortedChosenPerformances[i]].TimeTable) < len(chosenPerformanceInfos[sortedChosenPerformances[j]].TimeTable)
	})

	for _, name := range sortedChosenPerformances {
		info := chosenPerformanceInfos[name]
		var foundValidTime bool
		for _, t := range info.TimeTable {
			startTime, endTime, err := parseTimeAndEnd(t, info.Duration)
			if err != nil {
				continue
			}
			if !hasConflict(startTime, endTime) {
				arranged[name] = PerformanceTime{
					PerformanceName: name,
					StartTime:       startTime.Format("15:04"),
					EndTime:         endTime.Format("15:04"),
				}
				foundValidTime = true
				break
			}
		}
		if !foundValidTime {
			unsatisfied[name] = true
		}
	}

	sorted := make([]PerformanceTime, 0, len(arranged))
	for k := range arranged {
		sorted = append(sorted, arranged[k])
	}

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].EndTime < sorted[j].EndTime
	})

	unsatisfiedPerformances := make([]string, 0, len(unsatisfied))
	for k := range unsatisfied {
		unsatisfiedPerformances = append(unsatisfiedPerformances, k)
	}

	return &ArrangePerformancesResponse{
		ArrangedPerformances:    sorted,
		UnsatisfiedPerformances: unsatisfiedPerformances,
	}, nil
}

type PlanItem struct {
	ActivityType         ActivityType `json:"activity_type" jsonschema_description:"活动类型,enum:attraction,enum:performance,enum:restaurant,enum:other"`
	StartTime            string       `json:"start_time" jsonschema_description:"活动开始时间，格式为15:04"`
	PerformanceStartTime *string      `json:"performance_start_time" jsonschema_description:"如果为 performance 类型，是演出开始时间，格式为15:04. 否则忽略这个参数或传 NULL"`
	Duration             *int         `json:"duration" jsonschema_description:"如果为 performance 类型，是演出时长，如果为 attraction 类型，是实际玩的时长，单位为分钟，否则忽略这个参数或传 Null"`
	QueueTime            *int         `json:"queue_time" jsonschema_description:"排队时间，单位为分钟，attraction 类型必填，否则忽略这个参数或传 NULL。如果用了高速票，QueueTime使用0"`
	Location             string       `json:"location" jsonschema_description:"计划所在的区域，如果是 move 类型的计划，这里是目标区域。可以填写的区域列表为通过 list_locations 获取的清单"`
	ActivityName         string       `json:"activity_name,omitempty" jsonschema_description:"move 之外的其他类型的计划，这里是项目的准确名称，如 attraction，performance，restaurant 的准确名称"`
}

type PlanItemValidationResult struct {
	PlanItem PlanItem `json:"plan_item" jsonschema_description:"完整一日计划其中的一个计划项"`
	IsValid  bool     `json:"is_valid"`
	ErrMsg   string   `json:"err_msg"`
}

type ValidatePlanItemsRequest struct {
	PlanItems []PlanItem `json:"plan_items" jsonschema_description:"一个完整的一日日程计划，包含了所有的活动和移动计划，按时间先后顺序排列"`
}
type ValidatePlanItemsResponse struct {
	ValidationResults []PlanItemValidationResult `json:"validation_results" jsonschema_description:"每个计划项的验证结果，包含是否有效和错误信息，只包含有问题的计划项"`
}

func ValidatePlanItems(_ context.Context, in *ValidatePlanItemsRequest) (out *ValidatePlanItemsResponse, err error) {
	results := make([]PlanItemValidationResult, 0, len(in.PlanItems))

	for i, item := range in.PlanItems {
		result := PlanItemValidationResult{
			PlanItem: item,
			IsValid:  true,
			ErrMsg:   "",
		}

		switch item.ActivityType {
		case "表演", ActivityTypePerformance:
			item.ActivityType = ActivityTypePerformance
		case "游乐设施", ActivityTypeAttraction:
			item.ActivityType = ActivityTypeAttraction
		case "餐厅", ActivityTypeRestaurant:
			item.ActivityType = ActivityTypeRestaurant
		default:
			item.ActivityType = ActivityTypeOther
		}

		// 1. validate basic information is correct by query activity name, location, activityType, performance duration
		if item.ActivityType != ActivityTypeAttraction && item.ActivityType != ActivityTypePerformance && item.ActivityType != ActivityTypeRestaurant && item.ActivityType != ActivityTypeOther {
			result.IsValid = false
			result.ErrMsg = fmt.Sprintf("Invalid activity type %s for plan item %s", item.ActivityType, item.ActivityName)
		}

		if item.ActivityType == ActivityTypeAttraction && (item.QueueTime == nil || *item.QueueTime < 0) {
			result.IsValid = false
			result.ErrMsg = fmt.Sprintf("Queue time must be non-negative for attraction %s", item.ActivityName)
		}

		if item.ActivityType == ActivityTypePerformance {
			if item.Duration == nil || *item.Duration <= 0 {
				result.IsValid = false
				result.ErrMsg = fmt.Sprintf("Performance duration must be positive for performance %s", item.ActivityName)
			}
		}

		// 2. validate PerformanceStartTime is the same or after StartTime
		if item.ActivityType == ActivityTypePerformance {
			startTime, err := time.Parse("15:04", item.StartTime)
			if err != nil {
				result.IsValid = false
				result.ErrMsg = fmt.Sprintf("Invalid start time format for performance %s", item.ActivityName)
			} else {
				performanceStartTime, err := time.Parse("15:04", *item.PerformanceStartTime)
				if err != nil {
					result.IsValid = false
					result.ErrMsg = fmt.Sprintf("Invalid performance start time format for performance %s", item.ActivityName)
				} else if performanceStartTime.Before(startTime) {
					result.IsValid = false
					result.ErrMsg = fmt.Sprintf("Performance start time %s is before start time %s for performance %s", performanceStartTime, item.StartTime, item.ActivityName)
				}
			}
		}

		// 3. validate PerformanceStartTime + Duration is at least 10 minutes before the StartTime of next PlanItem, if there is a next PlanItem
		if item.ActivityType == ActivityTypePerformance && i < len(in.PlanItems)-1 {
			performanceStartTime, err := time.Parse("15:04", *item.PerformanceStartTime)
			if err != nil {
				result.IsValid = false
				result.ErrMsg = fmt.Sprintf("Invalid performance start time format for performance %s", item.ActivityName)
			} else {
				endTime := performanceStartTime.Add(time.Duration(*item.Duration) * time.Minute)
				nextStartTime, err := time.Parse("15:04", in.PlanItems[i+1].StartTime)
				if err != nil {
					result.IsValid = false
					result.ErrMsg = fmt.Sprintf("Invalid start time format for next plan item %s", in.PlanItems[i+1].ActivityName)
				} else if !endTime.Add(10 * time.Minute).Before(nextStartTime) {
					result.IsValid = false
					result.ErrMsg = fmt.Sprintf("Performance %s ends too close to the start time of next plan item %s", item.ActivityName, in.PlanItems[i+1].ActivityName)
				}
			}
		}

		// 4. if it's an attraction, StartTime + QueueTime + Duration should be before the StartTime of next PlanItem, if there is a next PlanItem
		if item.ActivityType == ActivityTypeAttraction && i < len(in.PlanItems)-1 {
			startTime, err := time.Parse("15:04", item.StartTime)
			if err != nil {
				result.IsValid = false
				result.ErrMsg = fmt.Sprintf("Invalid start time format for attraction %s", item.ActivityName)
			} else {
				endTime := startTime.Add(time.Duration(*item.QueueTime) * time.Minute).Add(time.Duration(*item.Duration) * time.Minute)
				nextStartTime, err := time.Parse("15:04", in.PlanItems[i+1].StartTime)
				if err != nil {
					result.IsValid = false
					result.ErrMsg = fmt.Sprintf("Invalid start time format for next plan item %s", in.PlanItems[i+1].ActivityName)
				} else if !endTime.Before(nextStartTime) {
					result.IsValid = false
					result.ErrMsg = fmt.Sprintf("Attraction %s's queue time ends too close to the start time of next plan item %s", item.ActivityName, in.PlanItems[i+1].ActivityName)
				}
			}
		}

		// 新增校验：上一个 PlanItem 的结束时间距离下一个 PlanItem 的开始时间不能超过半小时
		if i > 0 {
			prevItem := in.PlanItems[i-1]
			var prevEndTime, prevStartTime time.Time
			var parseErr error

			switch prevItem.ActivityType {
			case ActivityTypePerformance:
				prevStartTime, parseErr = time.Parse("15:04", *prevItem.PerformanceStartTime)
				if parseErr != nil {
					result.IsValid = false
					result.ErrMsg = fmt.Sprintf("Invalid performance start time format for previous plan item %s", prevItem.ActivityName)
					break
				}
				prevEndTime = prevStartTime.Add(time.Duration(*prevItem.Duration) * time.Minute)
			case ActivityTypeAttraction:
				prevStartTime, parseErr = time.Parse("15:04", prevItem.StartTime)
				if parseErr != nil {
					result.IsValid = false
					result.ErrMsg = fmt.Sprintf("Invalid start time format for previous attraction %s", prevItem.ActivityName)
					break
				}
				prevEndTime = prevStartTime.Add(time.Duration(*prevItem.QueueTime) * time.Minute).Add(10 * time.Minute)
			case ActivityTypeRestaurant:
				prevStartTime, parseErr = time.Parse("15:04", prevItem.StartTime)
				if parseErr != nil {
					result.IsValid = false
					result.ErrMsg = fmt.Sprintf("Invalid start time format for previous restaurant %s", prevItem.ActivityName)
					break
				}
				prevEndTime = prevStartTime.Add(45 * time.Minute)
			case ActivityTypeOther:
				prevStartTime, parseErr = time.Parse("15:04", prevItem.StartTime)
				if parseErr != nil {
					result.IsValid = false
					result.ErrMsg = fmt.Sprintf("Invalid start time format for previous move %s", prevItem.Location)
					break
				}
				prevEndTime = prevStartTime.Add(10 * time.Minute)
			}

			if parseErr == nil {
				currentStartTime, err := time.Parse("15:04", item.StartTime)
				if err != nil {
					result.IsValid = false
					result.ErrMsg = fmt.Sprintf("Invalid start time format for current plan item %s", item.ActivityName)
				} else if currentStartTime.Sub(prevEndTime) > 60*time.Minute {
					result.IsValid = false
					result.ErrMsg = fmt.Sprintf("The time gap between the end of previous plan item %s and the start of current plan item %s exceeds 60 minutes", prevItem.ActivityName, item.ActivityName)
				}
			}
		}

		if !result.IsValid {
			results = append(results, result)
		}
	}

	return &ValidatePlanItemsResponse{
		ValidationResults: results,
	}, nil
}
