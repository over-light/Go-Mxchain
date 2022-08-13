package groups

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestParseBoolUrlParam(t *testing.T) {
	c := &gin.Context{
		Request: &http.Request{
			URL: &url.URL{
				RawQuery: "a=true&b=false&c=foobar&d",
			},
		},
	}

	value, err := parseBoolUrlParam(c, "a")
	require.Nil(t, err)
	require.True(t, value)

	value, err = parseBoolUrlParam(c, "b")
	require.Nil(t, err)
	require.False(t, value)

	value, err = parseBoolUrlParam(c, "c")
	require.NotNil(t, err)
	require.False(t, value)

	value, err = parseBoolUrlParam(c, "d")
	require.Nil(t, err)
	require.False(t, value)

	value, err = parseBoolUrlParam(c, "e")
	require.Nil(t, err)
	require.False(t, value)
}

func TestParseUint32UrlParam(t *testing.T) {
	c := &gin.Context{
		Request: &http.Request{
			URL: &url.URL{
				RawQuery: "a=7&b=0&c=foobar&d=-1&e=4294967295&f=4294967296",
			},
		},
	}

	value, err := parseUint32UrlParam(c, "a")
	require.Nil(t, err)
	require.True(t, value.HasValue)
	require.Equal(t, uint32(7), value.Value)

	value, err = parseUint32UrlParam(c, "b")
	require.Nil(t, err)
	require.True(t, value.HasValue)
	require.Equal(t, uint32(0), value.Value)

	value, err = parseUint32UrlParam(c, "c")
	require.NotNil(t, err)
	require.False(t, value.HasValue)
	require.Equal(t, uint32(0), value.Value)

	value, err = parseUint32UrlParam(c, "d")
	require.NotNil(t, err)
	require.False(t, value.HasValue)
	require.Equal(t, uint32(0), value.Value)

	value, err = parseUint32UrlParam(c, "e")
	require.True(t, value.HasValue)
	require.Equal(t, uint32(0xffffffff), value.Value)

	value, err = parseUint32UrlParam(c, "f")
	require.NotNil(t, err)
	require.False(t, value.HasValue)
	require.Equal(t, uint32(0), value.Value)
}
