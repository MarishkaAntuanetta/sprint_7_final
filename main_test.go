package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCafeNegative(t *testing.T) {
	handler := http.HandlerFunc(mainHandle)

	requests := []struct {
		request string
		status  int
		message string
	}{
		{"/cafe", http.StatusBadRequest, "unknown city"},
		{"/cafe?city=omsk", http.StatusBadRequest, "unknown city"},
		{"/cafe?city=tula&count=na", http.StatusBadRequest, "incorrect count"},
	}
	for _, v := range requests {
		response := httptest.NewRecorder()
		req := httptest.NewRequest("GET", v.request, nil)
		handler.ServeHTTP(response, req)

		assert.Equal(t, v.status, response.Code)
		assert.Equal(t, v.message, strings.TrimSpace(response.Body.String()))
	}
}

func TestCafeWhenOk(t *testing.T) {
	handler := http.HandlerFunc(mainHandle)

	requests := []string{
		"/cafe?count=2&city=moscow",
		"/cafe?city=tula",
		"/cafe?city=moscow&search=ложка",
	}
	for _, v := range requests {
		response := httptest.NewRecorder()
		req := httptest.NewRequest("GET", v, nil)

		handler.ServeHTTP(response, req)

		assert.Equal(t, http.StatusOK, response.Code)
	}
}

func TestCafeCount(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(mainHandle))
	defer server.Close()

	city := "moscow"
	requests := []struct {
		count int
		want  int
	}{
		{0, 0},
		{1, 1},
		{2, 2},
		{100, -1},
	}

	for _, r := range requests {
		url := fmt.Sprintf("%s/cafe?city=%s&count=%d", server.URL, city, r.count)
		resp, err := http.Get(url)

		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		bodyBytes, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()

		body := string(bodyBytes)
		var cafes []string

		if body != "" && body != "," {
			cafes = strings.Split(body, ",")
		}

		if r.count == 100 {
			expectedMax := len(cafeList[city])
			if expectedMax > 100 {
				expectedMax = 100
			}
			assert.Equal(t, expectedMax, len(cafes), "Для count=100 ожидалось максимум 100 кафе")
		} else {
			assert.Equal(t, r.want, len(cafes), "Неверное количество кафе для count=%d", r.count)
		}
	}
}

func TestCafeSearch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(mainHandle))
	defer server.Close()

	city := "moscow"
	requests := []struct {
		search    string
		wantCount int
	}{
		{"фасоль", 0},
		{"кофе", 2},
		{"вилка", 1},
	}

	for _, r := range requests {
		url := fmt.Sprintf("%s/cafe?city=%s&search=%s", server.URL, city, r.search)
		resp, err := http.Get(url)

		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		bodyBytes, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()

		body := strings.TrimSpace(string(bodyBytes)) // убираем лишние пробелы/переводы
		var cafes []string

		if body != "" {
			cafes = strings.Split(body, ",")
		}

		assert.Equal(t, r.wantCount, len(cafes),
			"Неверное количество найденных кафе для search='%s'", r.search)

		for _, cafeName := range cafes {
			assert.True(t, strings.Contains(
				strings.ToLower(cafeName),
				strings.ToLower(r.search)),
				"Название кафе '%s' не содержит строки поиска '%s'",
				cafeName, r.search)
		}
	}
}
