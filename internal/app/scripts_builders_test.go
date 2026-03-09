package app

import "testing"

func TestRequireAuthToken(t *testing.T) {
	_, err := requireAuthToken(&runtimeConfig{authToken: "   "})
	if err == nil {
		t.Fatal("expected missing auth token error")
	}
	token, err := requireAuthToken(&runtimeConfig{authToken: " tok "})
	if err != nil || token != "tok" {
		t.Fatalf("unexpected token result: token=%q err=%v", token, err)
	}
}
