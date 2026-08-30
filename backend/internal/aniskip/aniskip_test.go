package aniskip

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchParsesAndNormalizes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"found":true,"results":[
			{"interval":{"startTime":12.3,"endTime":95.1},"skipType":"op"},
			{"interval":{"startTime":1350.0,"endTime":1410.5},"skipType":"ed"},
			{"interval":{"startTime":5,"endTime":5},"skipType":"recap"}
		]}`))
	}))
	defer srv.Close()

	oldC, oldAPI := httpClient, api
	httpClient, api = srv.Client(), srv.URL
	defer func() { httpClient, api = oldC, oldAPI }()

	got, err := Fetch(context.Background(), 12345, 1, 1441.0)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 markers, got %d: %+v", len(got), got)
	}
	if got[0].Type != "opening" || got[0].StartMs != 12300 || got[0].EndMs != 95100 {
		t.Errorf("op marker wrong: %+v", got[0])
	}
	if got[1].Type != "ending" || got[1].Name != "Ending" {
		t.Errorf("ed marker wrong: %+v", got[1])
	}
}

func TestFetchGuards(t *testing.T) {
	if m, _ := Fetch(context.Background(), 0, 1, 0); m != nil {
		t.Error("malID 0 should return nil")
	}
	if m, _ := Fetch(context.Background(), 1, 0, 0); m != nil {
		t.Error("episode 0 should return nil")
	}
}
