// +build integration

package main

import (
	"context"
	"net/http"
	"testing"

	"github.com/osbuild/osbuild-composer/internal/kojiapi"
	"github.com/stretchr/testify/require"
)

func TestKoji(t *testing.T) {
	client, err := kojiapi.NewClientWithResponses("http://127.0.0.1:8701/")
	if err != nil {
		panic(err)
	}

	response, err := client.PostComposeWithResponse(context.Background(), kojiapi.PostComposeJSONRequestBody{
		Distribution: "fedora-32",
		ImageRequests: []kojiapi.ImageRequest{
			{
				Architecture: "x86_64",
				ImageType:    "qcow2",
				Repositories: []kojiapi.Repository{
					{
						Baseurl: "http://download.fedoraproject.org/pub/fedora/linux/releases/32/Everything/x86_64/os/",
						Gpgkey:  "-----BEGIN PGP PUBLIC KEY BLOCK-----\n\nmQINBF1RVqsBEADWMBqYv/G1r4PwyiPQCfg5fXFGXV1FCZ32qMi9gLUTv1CX7rYy\nH4Inj93oic+lt1kQ0kQCkINOwQczOkm6XDkEekmMrHknJpFLwrTK4AS28bYF2RjL\nM+QJ/dGXDMPYsP0tkLvoxaHr9WTRq89A+AmONcUAQIMJg3JxXAAafBi2UszUUEPI\nU35MyufFt2ePd1k/6hVAO8S2VT72TxXSY7Ha4X2J0pGzbqQ6Dq3AVzogsnoIi09A\n7fYutYZPVVAEGRUqavl0th8LyuZShASZ38CdAHBMvWV4bVZghd/wDV5ev3LXUE0o\nitLAqNSeiDJ3grKWN6v0qdU0l3Ya60sugABd3xaE+ROe8kDCy3WmAaO51Q880ZA2\niXOTJFObqkBTP9j9+ZeQ+KNE8SBoiH1EybKtBU8HmygZvu8ZC1TKUyL5gwGUJt8v\nergy5Bw3Q7av520sNGD3cIWr4fBAVYwdBoZT8RcsnU1PP67NmOGFcwSFJ/LpiOMC\npZ1IBvjOC7KyKEZY2/63kjW73mB7OHOd18BHtGVkA3QAdVlcSule/z68VOAy6bih\nE6mdxP28D4INsts8w6yr4G+3aEIN8u0qRQq66Ri5mOXTyle+ONudtfGg3U9lgicg\nz6oVk17RT0jV9uL6K41sGZ1sH/6yTXQKagdAYr3w1ix2L46JgzC+/+6SSwARAQAB\ntDFGZWRvcmEgKDMyKSA8ZmVkb3JhLTMyLXByaW1hcnlAZmVkb3JhcHJvamVjdC5v\ncmc+iQI4BBMBAgAiBQJdUVarAhsPBgsJCAcDAgYVCAIJCgsEFgIDAQIeAQIXgAAK\nCRBsEwJtEslE0LdAD/wKdAMtfzr7O2y06/sOPnrb3D39Y2DXbB8y0iEmRdBL29Bq\n5btxwmAka7JZRJVFxPsOVqZ6KARjS0/oCBmJc0jCRANFCtM4UjVHTSsxrJfuPkel\nvrlNE9tcR6OCRpuj/PZgUa39iifF/FTUfDgh4Q91xiQoLqfBxOJzravQHoK9VzrM\nNTOu6J6l4zeGzY/ocj6DpT+5fdUO/3HgGFNiNYPC6GVzeiA3AAVR0sCyGENuqqdg\nwUxV3BIht05M5Wcdvxg1U9x5I3yjkLQw+idvX4pevTiCh9/0u+4g80cT/21Cxsdx\n7+DVHaewXbF87QQIcOAing0S5QE67r2uPVxmWy/56TKUqDoyP8SNsV62lT2jutsj\nLevNxUky011g5w3bc61UeaeKrrurFdRs+RwBVkXmtqm/i6g0ZTWZyWGO6gJd+HWA\nqY1NYiq4+cMvNLatmA2sOoCsRNmE9q6jM/ESVgaH8hSp8GcLuzt9/r4PZZGl5CvU\neldOiD221u8rzuHmLs4dsgwJJ9pgLT0cUAsOpbMPI0JpGIPQ2SG6yK7LmO6HFOxb\nAkz7IGUt0gy1MzPTyBvnB+WgD1I+IQXXsJbhP5+d+d3mOnqsd6oDM/grKBzrhoUe\noNadc9uzjqKlOrmrdIR3Bz38SSiWlde5fu6xPqJdmGZRNjXtcyJlbSPVDIloxw==\n=QWRO\n-----END PGP PUBLIC KEY BLOCK-----\n",
					},
				},
			},
		},
	})
	require.NoError(t, err)

	require.Equalf(t, http.StatusCreated, response.StatusCode(), "Error: got non-201 status. Full response: %v", string(response.Body))

	require.NotNil(t, response.JSON201)

	response2, err := client.GetComposeIdWithResponse(context.Background(), response.JSON201.Id)
	require.NoError(t, err)

	require.Equalf(t, response2.StatusCode(), 200, "Error: got non-200 status. Full response: %v", response2.Body)
}
