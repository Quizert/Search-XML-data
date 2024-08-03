package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
)

type Testcase struct {
	Name     string
	Request  SearchRequest
	Response SearchResponse
	IsError  bool
}

func sortID(users *[]User, orderBy int) {
	switch orderBy {
	case 0:
		return

	case 1:
		sort.Slice(*users, func(i, j int) bool {
			return (*users)[i].Id <= (*users)[j].Id
		})
	case -1:
		sort.Slice(*users, func(i, j int) bool {
			return (*users)[i].Id > (*users)[j].Id
		})
	}
}

func sortAge(users *[]User, orderBy int) {
	switch orderBy {
	case 0:
		return
	case 1:
		sort.Slice(*users, func(i, j int) bool {
			return (*users)[i].Age <= (*users)[j].Age
		})
	case -1:
		sort.Slice(*users, func(i, j int) bool {
			return (*users)[i].Age > (*users)[j].Age
		})
	}
}

func sortName(users *[]User, orderBy int) {
	switch orderBy {
	case 0:
		return

	case 1:
		sort.Slice(*users, func(i, j int) bool {
			return (*users)[i].Name <= (*users)[j].Name
		})
	case -1:
		sort.Slice(*users, func(i, j int) bool {
			return (*users)[i].Name > (*users)[j].Name
		})
	}
}

type Persons struct {
	Row []struct {
		Id        int    `xml:"id"`
		FirstName string `xml:"first_name"`
		LastName  string `xml:"last_name"`
		Age       int    `xml:"age"`
		About     string `xml:"about"`
		Gender    string `xml:"gender"`
	} `xml:"row"`
}

func SearchServer(w http.ResponseWriter, r *http.Request) {
	// Проверка метода запроса
	if r.Method != "GET" {
		http.Error(w, "Only GET method is allowed", http.StatusMethodNotAllowed)
		return
	}
	urlQuery := r.URL.Query()
	limit, _ := strconv.Atoi(urlQuery.Get("limit"))
	offset, _ := strconv.Atoi(urlQuery.Get("offset"))
	query := urlQuery.Get("query")
	orderField := urlQuery.Get("order_field")
	orderBy, _ := strconv.Atoi(urlQuery.Get("order_by"))

	persons := &Persons{}
	response := []User{}
	xmlRead, _ := os.ReadFile("dataset.xml")
	err := xml.Unmarshal(xmlRead, &persons)
	currentOffset := 0
	if err != nil {
		return
	}
	for _, person := range persons.Row {
		name := person.FirstName + " " + person.LastName

		if strings.Contains(name, query) || strings.Contains(person.About, query) {
			if len(response) == limit {
				break
			}
			if currentOffset < offset {
				currentOffset++
				continue
			}
			response = append(response, User{
				Id:     person.Id,
				Name:   name,
				Age:    person.Age,
				About:  person.About,
				Gender: person.Gender,
			})
		}

	}
	switch orderField {
	case "Id":
		sortID(&response, orderBy)
	case "Age":
		sortAge(&response, orderBy)
	case "Name", "":
		sortName(&response, orderBy)
	default:
		http.Error(w, "Invalid order_field", http.StatusBadRequest)
	}
	body, _ := json.Marshal(response)
	w.Write(body)
}

func TestServer(t *testing.T) {
	cases := []Testcase{
		{
			Name: "success",
			Request: SearchRequest{
				Limit:      3,
				Offset:     1,
				Query:      "al",
				OrderField: "Age",
				OrderBy:    1,
			},
			Response: SearchResponse{
				Users: []User{{Id: 1, Name: "Hilda Mayer", Age: 21, About: "Sit commodo consectetur minim amet ex. Elit aute mollit fugiat labore sint ipsum dolor cupidatat qui reprehenderit. Eu nisi in exercitation culpa sint aliqua nulla nulla proident eu. Nisi reprehenderit anim cupidatat dolor incididunt laboris mollit magna commodo ex. Cupidatat sit id aliqua amet nisi et voluptate voluptate commodo ex eiusmod et nulla velit.\n", Gender: "female"},
					{Id: 2, Name: "Brooks Aguilar", Age: 25, About: "Velit ullamco est aliqua voluptate nisi do. Voluptate magna anim qui cillum aliqua sint veniam reprehenderit consectetur enim. Laborum dolore ut eiusmod ipsum ad anim est do tempor culpa ad do tempor. Nulla id aliqua dolore dolore adipisicing.\n", Gender: "male"},
					{Id: 3, Name: "Everett Dillard", Age: 27, About: "Sint eu id sint irure officia amet cillum. Amet consectetur enim mollit culpa laborum ipsum adipisicing est laboris. Adipisicing fugiat esse dolore aliquip quis laborum aliquip dolore. Pariatur do elit eu nostrud occaecat.\n", Gender: "male"},
				}, NextPage: true,
			},
			IsError: false,
		},
		{
			Name: "limit < 0",
			Request: SearchRequest{
				Limit:      -3,
				Offset:     1,
				Query:      "al",
				OrderField: "Age",
				OrderBy:    1,
			},
			IsError: true,
		},
		{
			Name: "limit > 25",
			Request: SearchRequest{
				Limit:      27,
				Offset:     0,
				Query:      "Hilda Mayer",
				OrderField: "Age",
				OrderBy:    1,
			},
			Response: SearchResponse{
				Users:    []User{{Id: 1, Name: "Hilda Mayer", Age: 21, About: "Sit commodo consectetur minim amet ex. Elit aute mollit fugiat labore sint ipsum dolor cupidatat qui reprehenderit. Eu nisi in exercitation culpa sint aliqua nulla nulla proident eu. Nisi reprehenderit anim cupidatat dolor incididunt laboris mollit magna commodo ex. Cupidatat sit id aliqua amet nisi et voluptate voluptate commodo ex eiusmod et nulla velit.\n", Gender: "female"}},
				NextPage: false,
			},
			IsError: false,
		},
		{
			Name: "bad_request",
			Request: SearchRequest{
				Limit:      3,
				Offset:     0,
				Query:      "al",
				OrderField: "Lol",
				OrderBy:    1,
			},
			IsError: true,
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	client := &SearchClient{
		AccessToken: "aboba",
		URL:         ts.URL,
	}

	for _, tc := range cases {
		re, err := client.FindUsers(tc.Request)
		if tc.IsError {
			if err == nil {
				t.Error("Excepted error, got nil")
			}
		} else {
			if err != nil {
				t.Error("Unexpected error") //Здесь вывести какая ошибка через структуру?
			}
		}
		if !equalResponses(&tc.Response, re) {
			t.Errorf("expected response %+v, got %+v", tc.Response, re)
		}
	}

}

func equalResponses(a, b *SearchResponse) bool {
	if a == nil || b == nil {
		return true
	}
	if a.NextPage != b.NextPage {
		return false
	}
	if len(a.Users) != len(b.Users) {
		return false
	}
	for i := range a.Users {
		if a.Users[i] != b.Users[i] {
			if a.Users[i].Age != b.Users[i].Age {
				fmt.Println(a.Users[i].About)
				fmt.Println(b.Users[i].About)
			}
			return false
		}
	}
	return true
}
