// Code generated by goa v3.4.2, DO NOT EDIT.
//
// reporter HTTP client CLI support package
//
// Command:
// $ goa gen github.com/lbryio/lbrytv/apps/watchman/design -o apps/watchman

package client

import (
	"encoding/json"
	"fmt"
	"unicode/utf8"

	reporter "github.com/lbryio/lbrytv/apps/watchman/gen/reporter"
	goa "goa.design/goa/v3/pkg"
)

// BuildAddPayload builds the payload for the reporter add endpoint from CLI
// flags.
func BuildAddPayload(reporterAddBody string) (*reporter.PlaybackReport, error) {
	var err error
	var body AddRequestBody
	{
		err = json.Unmarshal([]byte(reporterAddBody), &body)
		if err != nil {
			return nil, fmt.Errorf("invalid JSON for body, \nerror: %s, \nexample of valid JSON:\n%s", err, "'{\n      \"client_rate\": 1906586091,\n      \"device\": \"adr\",\n      \"dur\": 54906,\n      \"format\": \"hls\",\n      \"player\": \"sg-p2\",\n      \"position\": 1152411017,\n      \"rebuf_count\": 268663686,\n      \"rebuf_duration\": 40307,\n      \"rel_position\": 99,\n      \"t\": \"Fri, 23 Dec 1983 15:20:34 UTC\",\n      \"url\": \"what\",\n      \"user_id\": 1143315912\n   }'")
		}
		if utf8.RuneCountInString(body.URL) > 512 {
			err = goa.MergeErrors(err, goa.InvalidLengthError("body.url", body.URL, utf8.RuneCountInString(body.URL), 512, false))
		}
		if body.Dur < 0 {
			err = goa.MergeErrors(err, goa.InvalidRangeError("body.dur", body.Dur, 0, true))
		}
		if body.Dur > 60000 {
			err = goa.MergeErrors(err, goa.InvalidRangeError("body.dur", body.Dur, 60000, false))
		}
		if body.Position < 0 {
			err = goa.MergeErrors(err, goa.InvalidRangeError("body.position", body.Position, 0, true))
		}
		if body.RelPosition < 0 {
			err = goa.MergeErrors(err, goa.InvalidRangeError("body.rel_position", body.RelPosition, 0, true))
		}
		if body.RelPosition > 100 {
			err = goa.MergeErrors(err, goa.InvalidRangeError("body.rel_position", body.RelPosition, 100, false))
		}
		if body.RebufCount < 0 {
			err = goa.MergeErrors(err, goa.InvalidRangeError("body.rebuf_count", body.RebufCount, 0, true))
		}
		if body.RebufDuration < 0 {
			err = goa.MergeErrors(err, goa.InvalidRangeError("body.rebuf_duration", body.RebufDuration, 0, true))
		}
		if body.RebufDuration > 60000 {
			err = goa.MergeErrors(err, goa.InvalidRangeError("body.rebuf_duration", body.RebufDuration, 60000, false))
		}
		if !(body.Format == "stb" || body.Format == "hls") {
			err = goa.MergeErrors(err, goa.InvalidEnumValueError("body.format", body.Format, []interface{}{"stb", "hls"}))
		}
		if utf8.RuneCountInString(body.Player) > 64 {
			err = goa.MergeErrors(err, goa.InvalidLengthError("body.player", body.Player, utf8.RuneCountInString(body.Player), 64, false))
		}
		if !(body.Device == "ios" || body.Device == "adr" || body.Device == "web") {
			err = goa.MergeErrors(err, goa.InvalidEnumValueError("body.device", body.Device, []interface{}{"ios", "adr", "web"}))
		}
		err = goa.MergeErrors(err, goa.ValidateFormat("body.t", body.T, goa.FormatRFC1123))

		if err != nil {
			return nil, err
		}
	}
	v := &reporter.PlaybackReport{
		URL:           body.URL,
		Dur:           body.Dur,
		Position:      body.Position,
		RelPosition:   body.RelPosition,
		RebufCount:    body.RebufCount,
		RebufDuration: body.RebufDuration,
		Format:        body.Format,
		Player:        body.Player,
		UserID:        body.UserID,
		ClientRate:    body.ClientRate,
		Device:        body.Device,
		T:             body.T,
	}

	return v, nil
}
