package main

import (
	"context"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"testing"

	appbsky "github.com/bluesky-social/indigo/api/bsky"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebProfileRSS(t *testing.T) {
	tests := []struct {
		name           string
		ident          string
		profile        *appbsky.ActorProfile
		feedResponse   *appbsky.FeedGetAuthorFeed_Output
		expectedStatus int
		expectedTitle  string
		wantErr       bool
	}{
		{
			name:  "valid handle with public profile",
			ident: "test.bsky.app",
			profile: &appbsky.ActorProfile{
				Did:         "did:plc:testuser123",
				Handle:      "test.bsky.app",
				DisplayName: stringPtr("Test User"),
				Description: stringPtr("Test Description"),
			},
			feedResponse: &appbsky.FeedGetAuthorFeed_Output{
				Feed: []*appbsky.FeedDefs_FeedViewPost{
					{
						Post: &appbsky.FeedDefs_PostView{
							Uri:    "at://did:plc:testuser123/app.bsky.feed.post/1234",
							Author: &appbsky.ActorDefs_ProfileViewBasic{Did: "did:plc:testuser123"},
							Record: &appbsky.FeedDefs_PostRecord{
								Val: &appbsky.FeedPost{
									Text:      "Test post content",
									CreatedAt: "2024-01-19T12:00:00Z",
								},
							},
						},
					},
				},
			},
			expectedStatus: http.StatusOK,
			expectedTitle:  "@test.bsky.app - Test User",
			wantErr:       false,
		},
		{
			name:  "private profile",
			ident: "private.bsky.app",
			profile: &appbsky.ActorProfile{
				Did:    "did:plc:private123",
				Handle: "private.bsky.app",
				Labels: []*appbsky.Label{
					{
						Src: "did:plc:private123",
						Val: "!no-unauthenticated",
					},
				},
			},
			expectedStatus: http.StatusForbidden,
			wantErr:       true,
		},
		{
			name:           "invalid handle",
			ident:         "invalid@handle",
			expectedStatus: http.StatusBadRequest,
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/profile/"+tt.ident+"/rss", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("ident")
			c.SetParamValues(tt.ident)

			// Create mock server with test data
			srv := &Server{
				xrpcc: &mockXRPCC{
					profile:      tt.profile,
					feedResponse: tt.feedResponse,
				},
			}

			// Execute test
			err := srv.WebProfileRSS(c)

			// Verify results
			if tt.wantErr {
				assert.Error(t, err)
				if he, ok := err.(*echo.HTTPError); ok {
					assert.Equal(t, tt.expectedStatus, he.Code)
				}
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, rec.Code)

			if tt.expectedStatus == http.StatusOK {
				var feed rss
				err = xml.Unmarshal(rec.Body.Bytes(), &feed)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedTitle, feed.Title)
				assert.Equal(t, "2.0", feed.Version)
			}
		})
	}
}

// Mock XRPC client for testing
type mockXRPCC struct {
	profile      *appbsky.ActorProfile
	feedResponse *appbsky.FeedGetAuthorFeed_Output
}

func (m *mockXRPCC) Do(ctx context.Context, op string, input, output interface{}) error {
	switch op {
	case "app.bsky.actor.getProfile":
		*(output.(*appbsky.ActorProfile)) = *m.profile
	case "app.bsky.feed.getAuthorFeed":
		*(output.(*appbsky.FeedGetAuthorFeed_Output)) = *m.feedResponse
	}
	return nil
}

// Helper function for string pointers
func stringPtr(s string) *string {
	return &s
}