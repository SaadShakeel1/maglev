package restapi

import (
	"fmt"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"maglev.onebusaway.org/internal/clock"
)

// TestStopsForRoutePreservesDateAcrossSkippedMidnight verifies that date
// filtering does not select yesterday when local midnight does not exist.
func TestStopsForRoutePreservesDateAcrossSkippedMidnight(t *testing.T) {
	for _, tc := range []struct{ zone, date string }{
		{"America/Santiago", "2026-09-06"},
		{"America/Havana", "2026-03-08"},
		{"UTC", "2026-09-06"},
	} {
		t.Run(tc.zone, func(t *testing.T) {
			loc, err := time.LoadLocation(tc.zone)
			require.NoError(t, err)
			day, err := time.Parse("2006-01-02", tc.date)
			require.NoError(t, err)
			noon := time.Date(day.Year(), day.Month(), day.Day(), 12, 0, 0, 0, loc)
			files := map[string]string{
				"agency.txt": "agency_id,agency_name,agency_url,agency_timezone\n" +
					"A,Date Transit,https://example.com," + tc.zone + "\n",
				"routes.txt": "route_id,agency_id,route_short_name,route_long_name,route_type\n" +
					"R,A,1,Daily Route,3\n",
				"calendar.txt": "service_id,monday,tuesday,wednesday,thursday,friday,saturday,sunday,start_date,end_date\n" +
					"today,0,0,0,0,0,0,1,20260101,20261231\n" +
					"yesterday,0,0,0,0,0,1,0,20260101,20261231\n",
				"stops.txt": "stop_id,stop_name,stop_lat,stop_lon\n" +
					"today1,Today First,40,-74\ntoday2,Today Last,40.01,-74\n" +
					"yesterday1,Yesterday First,40.02,-74\nyesterday2,Yesterday Last,40.03,-74\n",
				"trips.txt": "route_id,service_id,trip_id,trip_headsign,direction_id\n" +
					"R,today,today-trip,Today,0\nR,yesterday,yesterday-trip,Yesterday,0\n",
				"stop_times.txt": "trip_id,arrival_time,departure_time,stop_id,stop_sequence\n" +
					"today-trip,12:00:00,12:00:00,today1,1\ntoday-trip,12:10:00,12:10:00,today2,2\n" +
					"yesterday-trip,12:00:00,12:00:00,yesterday1,1\nyesterday-trip,12:10:00,12:10:00,yesterday2,2\n",
			}
			api := createTestApiWithGTFSFixture(t, clock.NewMockClock(noon), "skipped-midnight.zip", files)
			for _, param := range []struct{ name, value string }{
				{"calendar_date", tc.date},
				{"epoch_millis", strconv.FormatInt(noon.UnixMilli(), 10)},
			} {
				t.Run(param.name, func(t *testing.T) {
					endpoint := fmt.Sprintf("/api/where/stops-for-route/A_R.json?key=TEST&time=%s&includePolylines=false", param.value)
					resp, model := callAPIHandler[StopsForRouteResponse](t, api, endpoint)
					require.Equal(t, http.StatusOK, resp.StatusCode)
					require.Equal(t, 200, model.Code)
					assert.ElementsMatch(t, []string{"A_today1", "A_today2"}, model.Data.Entry.StopIds)
				})
			}
		})
	}
}
