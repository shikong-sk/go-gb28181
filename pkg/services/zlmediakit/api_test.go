package zlmediakit

import (
	"encoding/json"
	"fmt"
	"git.skcks.cn/Shikong/go-gb28181/pkg/services/zlmediakit/types"
	"github.com/duke-git/lancet/v2/convertor"
	"github.com/duke-git/lancet/v2/netutil"
	"github.com/go-resty/resty/v2"
	"testing"
)

func TestZLMediaKit(t *testing.T) {
	SetupZLMediaKitService(&Config{
		Id:     "amrWMKmbKqoBjRQ9",
		Url:    "http://10.10.10.200:5081",
		Secret: "4155cca6-2f9f-11ee-85e6-8de4ce2e7333",
	})

	resp, err := zLMediaKitService.client.R().Get("/index/api/getMediaList")
	if err != nil {
		t.Fatal(err)
	}
	data := &types.GetMediaListResp{}
	_ = json.Unmarshal(resp.Body(), &data)
	t.Logf("%+v\n", data)
	for _, datum := range data.Data {
		t.Logf("%+v\n", datum)
	}
}

func TestZLMediaKit_API(t *testing.T) {
	SetupZLMediaKitService(&Config{
		Id:     "amrWMKmbKqoBjRQ9",
		Url:    "http://10.10.10.200:5081",
		Secret: "4155cca6-2f9f-11ee-85e6-8de4ce2e7333",
	})

	versionResp, err := zLMediaKitService.client.R().Get("/index/api/version")
	printRequest(versionResp, err)

	version := &types.VersionResp{}
	_ = json.Unmarshal(versionResp.Body(), &version)
	fmt.Printf("Version: %+v\n\n", version)

	serverConfigResp, err := zLMediaKitService.client.R().Get("/index/api/getServerConfig")
	printRequest(serverConfigResp, err)
	serverConfig := &types.ServerConfigResp{}
	_ = json.Unmarshal(serverConfigResp.Body(), &serverConfig)
	fmt.Printf("ServerConfig: %+v\n\n", serverConfig)

	printRequest(zLMediaKitService.client.R().Get("/index/api/getMediaList"))
}

func printRequest(resp *resty.Response, err error) {
	fmt.Println("Request URL:", resp.Request.URL)
	printResponse(resp, err)
	printResponseTrace(resp)
}

func printResponse(resp *resty.Response, err error) {
	fmt.Println("Response Info:")
	fmt.Println("  Error      :", err)
	fmt.Println("  Status Code:", resp.StatusCode())
	fmt.Println("  Status     :", resp.Status())
	fmt.Println("  Proto      :", resp.Proto())
	fmt.Println("  Time       :", resp.Time())
	fmt.Println("  Received At:", resp.ReceivedAt())
	fmt.Println("  Body       :\n", resp)
	fmt.Println()
}

func printResponseTrace(resp *resty.Response) {
	fmt.Println("Request Trace Info:")
	ti := resp.Request.TraceInfo()
	fmt.Println("  DNSLookup     :", ti.DNSLookup)
	fmt.Println("  ConnTime      :", ti.ConnTime)
	fmt.Println("  TCPConnTime   :", ti.TCPConnTime)
	fmt.Println("  TLSHandshake  :", ti.TLSHandshake)
	fmt.Println("  ServerTime    :", ti.ServerTime)
	fmt.Println("  ResponseTime  :", ti.ResponseTime)
	fmt.Println("  TotalTime     :", ti.TotalTime)
	fmt.Println("  IsConnReused  :", ti.IsConnReused)
	fmt.Println("  IsConnWasIdle :", ti.IsConnWasIdle)
	fmt.Println("  ConnIdleTime  :", ti.ConnIdleTime)
	fmt.Println("  RequestAttempt:", ti.RequestAttempt)
	fmt.Println("  RemoteAddr    :", ti.RemoteAddr.String())
	fmt.Println()
}

func TestConvertor(t *testing.T) {
	m, _ := convertor.StructToMap(&types.OpenRtpServerReq{
		Port:     5050,
		TcpMode:  0,
		StreamId: "123456",
	})

	t.Log(m)

	resp, _ := resty.New().R().
		SetQueryString(netutil.ConvertMapToQueryString(m)).Get("/index/api/openRtpServer")

	t.Log(resp.Request.URL)
}
