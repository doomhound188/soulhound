package audio

import (
	"testing"
)

func TestYouTubeProvider(t *testing.T) {
	// Test with no API key (should return mock data)
	yt := NewYouTubeProvider("")
	
	results, err := yt.Search("test query")
	if err != nil {
		t.Errorf("Search failed: %v", err)
	}
	
	if len(results) == 0 {
		t.Error("Expected mock results when no API key provided")
	}
	
	// Check that mock results have required fields
	for _, result := range results {
		if result.ID == "" {
			t.Error("Expected non-empty ID in mock result")
		}
		if result.Title == "" {
			t.Error("Expected non-empty title in mock result")
		}
		if result.Artist == "" {
			t.Error("Expected non-empty artist in mock result")
		}
	}
}

func TestYouTubeStreamURL(t *testing.T) {
	yt := NewYouTubeProvider("")
	
	// Test with mock ID
	url, err := yt.GetStreamURL("mock_test_123")
	if err != nil {
		t.Errorf("GetStreamURL failed for mock ID: %v", err)
	}
	if url != "mock_test_123" {
		t.Errorf("Expected mock ID to be returned as-is, got: %s", url)
	}
	
	// Test with real YouTube ID
	url, err = yt.GetStreamURL("dQw4w9WgXcQ")
	if err != nil {
		t.Errorf("GetStreamURL failed for YouTube ID: %v", err)
	}
	if url != "dQw4w9WgXcQ" {
		t.Errorf("Expected YouTube ID to be returned as-is, got: %s", url)
	}
	
	// Test with empty ID
	_, err = yt.GetStreamURL("")
	if err == nil {
		t.Error("Expected error for empty ID")
	}
}

func TestYouTubeRecommendations(t *testing.T) {
	yt := NewYouTubeProvider("")
	
	// Test with known genre
	recs, err := yt.GetRecommendations("pop")
	if err != nil {
		t.Errorf("GetRecommendations failed: %v", err)
	}
	if len(recs) == 0 {
		t.Error("Expected recommendations for pop genre")
	}
	
	// Test with unknown genre
	recs, err = yt.GetRecommendations("unknown")
	if err != nil {
		t.Errorf("GetRecommendations failed for unknown genre: %v", err)
	}
	if len(recs) == 0 {
		t.Error("Expected default recommendations for unknown genre")
	}
}

func TestSpotifyProvider(t *testing.T) {
	// Test with no API key (should return mock data)
	sp := NewSpotifyProvider("")
	
	results, err := sp.Search("test query")
	if err != nil {
		t.Errorf("Search failed: %v", err)
	}
	
	if len(results) == 0 {
		t.Error("Expected mock results when no API key provided")
	}
	
	// Check that mock results have required fields
	for _, result := range results {
		if result.ID == "" {
			t.Error("Expected non-empty ID in mock result")
		}
		if result.Title == "" {
			t.Error("Expected non-empty title in mock result")
		}
		if result.Artist == "" {
			t.Error("Expected non-empty artist in mock result")
		}
	}
}

func TestSpotifyStreamURL(t *testing.T) {
	sp := NewSpotifyProvider("")
	
	// Test with mock ID
	url, err := sp.GetStreamURL("spotify_mock_test_123")
	if err != nil {
		t.Errorf("GetStreamURL failed for mock ID: %v", err)
	}
	if url != "spotify_mock_test_123" {
		t.Errorf("Expected mock ID to be returned as-is, got: %s", url)
	}
	
	// Test with real Spotify ID
	url, err = sp.GetStreamURL("4iV5W9uYEdYUVa79Axb7Rh")
	if err != nil {
		t.Errorf("GetStreamURL failed for Spotify ID: %v", err)
	}
	if url != "4iV5W9uYEdYUVa79Axb7Rh" {
		t.Errorf("Expected Spotify ID to be returned as-is, got: %s", url)
	}
	
	// Test with empty ID
	_, err = sp.GetStreamURL("")
	if err == nil {
		t.Error("Expected error for empty ID")
	}
}

func TestSpotifyRecommendations(t *testing.T) {
	sp := NewSpotifyProvider("")
	
	// Test with known genre
	recs, err := sp.GetRecommendations("rock")
	if err != nil {
		t.Errorf("GetRecommendations failed: %v", err)
	}
	if len(recs) == 0 {
		t.Error("Expected recommendations for rock genre")
	}
	
	// Test with unknown genre
	recs, err = sp.GetRecommendations("unknown")
	if err != nil {
		t.Errorf("GetRecommendations failed for unknown genre: %v", err)
	}
	if len(recs) == 0 {
		t.Error("Expected default recommendations for unknown genre")
	}
}