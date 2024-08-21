package arweave

import (
	"encoding/json"
	"os"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ybbus/jsonrpc"
)

func TestReplaceAssetUrls(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	f, err := os.ReadFile("../../claims.json")
	require.NoError(err)
	var resp jsonrpc.RPCResponse
	require.NoError(json.Unmarshal(f, &resp))
	result, err := ReplaceAssetUrls("http://odycdn.com", resp.Result, "items", "value.thumbnail.url")
	require.NoError(err)

	out, err := json.MarshalIndent(result, "", "  ")
	require.NoError(err)
	re := regexp.MustCompile(`http://odycdn.com/explore/\w{64}\?filename=\w{64}\.webp`)
	matches := re.FindAllString(string(out), -1)
	assert.Equal(2, len(matches))
}

func TestReplaceAssetUrl(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	f, err := os.ReadFile("../../resolve.json")
	require.NoError(err)
	var resp jsonrpc.RPCResponse
	require.NoError(json.Unmarshal(f, &resp))
	result, err := ReplaceAssetUrl("http://odycdn.com", resp.Result.(map[string]any)["lbry://@MySillyReactions#d1ae6a9097b44691d318a5bfc6dc1240311c75e2"], "value.thumbnail.url")
	require.NoError(err)

	out, err := json.MarshalIndent(result, "", "  ")
	require.NoError(err)
	assert.Regexp(`http://odycdn.com/explore/\w{64}\?filename=\w{64}\.jpg`, string(out))
}