/*
 * Copyright 2024 CloudWeGo Authors
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
	"strings"
)

// fake service 模拟的后端服务的 service
// 提供 QueryDishes, QueryRestaurants 两个方法.
var restService = &fakeService{
	repo: database,
}

// fake database.
var database = &restaurantDatabase{
	restaurantByID:        make(map[string]restaurantDataItem),
	restaurantsByLocation: make(map[string][]restaurantDataItem),
}

func init() {
	// prepare database
	restData := getData()
	for location, rests := range restData {
		for _, rest := range rests {
			database.restaurantByID[rest.ID] = rest
			database.restaurantsByLocation[location] = append(database.restaurantsByLocation[location], rest)
		}
	}
}

// ====== fake service ======
type fakeService struct {
	repo *restaurantDatabase
}

// QueryRestaurants 查询一个 location 的餐厅列表.
func (ft *fakeService) QueryRestaurants(ctx context.Context, in *QueryRestaurantsParam) (out []Restaurant, err error) {
	rests, err := ft.repo.GetRestaurantsByLocation(ctx, in.Location, in.Topn)
	if err != nil {
		return nil, err
	}

	res := make([]Restaurant, 0, len(rests))
	for _, rest := range rests {
		res = append(res, Restaurant{
			ID:    rest.ID,
			Name:  rest.Name,
			Place: rest.Place,
			Score: rest.Score,
		})
	}

	return res, nil
}

// QueryDishes 根据餐厅的 id, 查询餐厅的菜品列表.
func (ft *fakeService) QueryDishes(ctx context.Context, in *QueryDishesParam) (res []Dish, err error) {
	dishes, err := ft.repo.GetDishesByRestaurant(ctx, in.RestaurantID, in.Topn)
	if err != nil {
		return nil, err
	}

	res = make([]Dish, 0, len(dishes))
	for _, dish := range dishes {
		res = append(res, Dish(dish))
	}

	return res, nil
}

type restaurantDishDataItem struct {
	Name  string `json:"name"`
	Desc  string `json:"desc"`
	Price int    `json:"price"`
	Score int    `json:"score"`
}

type restaurantDataItem struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Desc  string `json:"desc"`
	Place string `json:"place"`
	Score int    `json:"score"` // 0 - 10

	Dishes []restaurantDishDataItem `json:"dishes"` // 餐厅中的菜
}

type restaurantDatabase struct {
	restaurantByID        map[string]restaurantDataItem   // id => restaurantDataItem
	restaurantsByLocation map[string][]restaurantDataItem // location => []restaurantDataItem
}

func (rd *restaurantDatabase) GetRestaurantsByLocation(ctx context.Context, location string, topn int) ([]restaurantDataItem, error) {
	for locationName, rests := range rd.restaurantsByLocation {
		if strings.Contains(locationName, location) || strings.Contains(location, locationName) {

			res := make([]restaurantDataItem, 0, len(rests))
			for i := 0; i < topn && i < len(rests); i++ {
				res = append(res, rests[i])
			}

			return res, nil
		}
	}

	return nil, fmt.Errorf("location %s not found", location)
}

func (rd *restaurantDatabase) GetDishesByRestaurant(ctx context.Context, restaurantID string, topn int) ([]restaurantDishDataItem, error) {
	rest, ok := rd.restaurantByID[restaurantID]
	if !ok {
		return nil, fmt.Errorf("restaurant %s not found", restaurantID)
	}

	res := make([]restaurantDishDataItem, 0, len(rest.Dishes))

	for i := 0; i < topn && i < len(rest.Dishes); i++ {
		res = append(res, rest.Dishes[i])
	}

	return res, nil
}

func getData() map[string][]restaurantDataItem {
	return map[string][]restaurantDataItem{
		"北京": {
			{
				ID:    "1001",
				Name:  "云边小馆",
				Place: "北京",
				Desc:  "这个是云边小馆, 在北京, 口味多种多样",
				Score: 3,
				Dishes: []restaurantDishDataItem{
					{
						Name:  "红烧肉",
						Desc:  "一块红烧肉",
						Price: 20,
						Score: 8,
					},
					{
						Name:  "清泉牛肉",
						Desc:  "很多的水煮牛肉",
						Price: 50,
						Score: 8,
					},
					{
						Name:  "清炒小南瓜",
						Desc:  "炒的糊糊的南瓜",
						Price: 5,
						Score: 5,
					},
					{
						Name:  "韩式辣白菜",
						Desc:  "这可是开过光的辣白菜，好吃得很",
						Price: 20,
						Score: 9,
					},
					{
						Name:  "酸辣土豆丝",
						Desc:  "酸酸辣辣的土豆丝",
						Price: 10,
						Score: 9,
					},
					{
						Name:  "酸辣粉",
						Desc:  "酸酸辣辣的粉",
						Price: 5,
					},
				},
			},
			{
				ID:    "1002",
				Name:  "聚福轩食府",
				Place: "北京",
				Desc:  "北京的聚福轩食府, 很多档口, 等你来探索",
				Score: 5,
				Dishes: []restaurantDishDataItem{
					{
						Name:  "红烧排骨",
						Desc:  "一块一块的排骨",
						Price: 43,
						Score: 7,
					},
					{
						Name:  "大刀回锅肉",
						Desc:  "经典的回锅肉, 肉很大",
						Price: 40,
						Score: 8,
					},
					{
						Name:  "火辣辣的吻",
						Desc:  "凉拌猪嘴，口味辣而不腻",
						Price: 60,
						Score: 9,
					},
					{
						Name:  "辣椒拌皮蛋",
						Desc:  "擂椒皮蛋，下饭的神器",
						Price: 15,
						Score: 8,
					},
				},
			},
			{
				ID:    "1003",
				Name:  "花影食舍",
				Place: "上海",
				Desc:  "非常豪华的花影食舍, 好吃不贵",
				Score: 10,
				Dishes: []restaurantDishDataItem{
					{
						Name:  "超级红烧肉",
						Desc:  "非常红润的一块红烧肉",
						Price: 30,
						Score: 9,
					},
					{
						Name:  "超级北京烤肉",
						Desc:  "卷好了的烤鸭，配上酱汁",
						Price: 60,
						Score: 9,
					},
					{
						Name:  "超级大白菜",
						Desc:  "就是炒的水水的大白菜",
						Price: 8,
						Score: 8,
					},
				},
			},
		},
		"上海": {
			{
				ID:    "2001",
				Name:  "鸿宾雅膳楼",
				Place: "上海",
				Desc:  "这个是鸿宾雅膳楼, 在上海, 口味多种多样",
				Score: 3,
				Dishes: []restaurantDishDataItem{
					{
						Name:  "糖醋西红柿",
						Desc:  "酸酸甜甜就是一个西红柿",
						Price: 80,
						Score: 5,
					},
					{
						Name:  "糖渍🐟",
						Desc:  "加了挺多糖的鱼，和醋鱼齐名",
						Price: 99,
						Score: 6,
					},
				},
			},
			{
				ID:    "2002",
				Name:  "饭醉团伙根据地",
				Desc:  "专注糖醋口味，你值得拥有",
				Place: "上海",
				Score: 5,
				Dishes: []restaurantDishDataItem{
					{
						Name:  "糖醋西瓜瓤",
						Desc:  "糖醋味，嘎嘣脆",
						Price: 69,
						Score: 7,
					},
					{
						Name:  "糖醋大包子",
						Desc:  "和天津狗不理齐名",
						Price: 99,
						Score: 4,
					},
				},
			},
			{
				ID:    "2010",
				Name:  "好吃到跺 jiojio 餐馆",
				Desc:  "这个是好吃到跺 jiojio 餐馆, 藏在一个你找不到的位置, 只等待有缘人来探索, 口味以川菜为主, 辣椒、花椒 大把大把放.",
				Place: "它在它不在的地方",
				Score: 10,
				Dishes: []restaurantDishDataItem{
					{
						Name:  "无敌香辣虾🦞",
						Desc:  "香香香香香香香香香香",
						Price: 199,
						Score: 9,
					},
					{
						Name:  "超级大火锅🍲",
						Desc:  "有很多辣椒和醪糟的火锅，可以煮东西，比如苹果🍌",
						Price: 198,
						Score: 9,
					},
				},
			},
		},
	}
}
