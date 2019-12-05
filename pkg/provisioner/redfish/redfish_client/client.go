package redfish_client

import (
	"fmt"
	"net/http"
	"github.com/imroc/req"
	"crypto/tls"
	"crypto/md5"
	"encoding/hex"
	"encoding/base64"
	"strings"
	"github.com/metal3-io/baremetal-operator/pkg/bmc"

)

type RedfishClient struct {
	Name 		string
	AuthType    string
	BmcCred		bmc.Credentials
	BaseURL		string
	Header 		req.Header
	HttpClient  *req.Req
}


func New(base_url string, bmcCred  bmc.Credentials) (RedfishClient) {

	https_base_url := strings.Replace(base_url, "redfish", "https", 1)

	//Initialize Redfish Client
	redfish_client := RedfishClient {
		Name:       "RedfishClient",
		AuthType:   "Basic",
		BaseURL:    https_base_url,
		BmcCred:  bmcCred,
	}
	client := GetClient()

	//Set Proper Authorization and Other Headers
	bmc_username := strings.TrimSuffix(redfish_client.BmcCred.Username, "\n") 
	bmc_password := strings.TrimSuffix(redfish_client.BmcCred.Password, "\n")  
	b64_encoded_cred := redfish_client.EncodeString(bmc_username + ":" + bmc_password)
	auth_type := redfish_client.AuthType
	redfish_client.SetHeader("Authorization", auth_type + " " + b64_encoded_cred)
	redfish_client.SetHeader("Accept", "application/json")
	redfish_client.SetHeader("Content-Type","application/json" )

	
	redfish_client.HttpClient = client
	return redfish_client
}

func GetClient() (*req.Req) {

	Req := req.New()
	trans, _ := Req.Client().Transport.(*http.Transport)
	trans.MaxIdleConns = 20
	trans.DisableKeepAlives = true
	trans.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	return Req
}


func (redfishClient RedfishClient) EncodeString(data string) (string) {

	return base64.StdEncoding.EncodeToString([]byte(data))
}


func (redfishClient *RedfishClient) SetHeader(key string, value string) (req.Header) {

	if redfishClient.Header == nil {
		redfishClient.Header = req.Header{}
	}
	redfishClient.Header[key] = value
	return redfishClient.Header
}

func (redfishClient RedfishClient) SystemURL(parts ...string) string {

	return redfishClient.BaseURL + "/Systems/System.Embedded.1/" + strings.Join(parts, "/")
}


func (redfishClient RedfishClient) ManagerURL(parts ...string) string {
	
	return redfishClient.BaseURL + "/Managers/iDRAC.Embedded.1/" + strings.Join(parts, "/")
}

func (redfishClient RedfishClient) EventURL(parts ...string) string {

	return redfishClient.BaseURL + "/EventService/" + strings.Join(parts, "/")
}





func (redfishClient RedfishClient) GetVirtualMediaStatus() (bool) {

	endpoint := redfishClient.ManagerURL("VirtualMedia","CD")
	header := redfishClient.Header
	r, err := redfishClient.HttpClient.Get(endpoint, header)
	
	res := CheckErrorAndReturn(r,err)
	var data map[string]interface{}
	res.ToJSON(&data)       // response => struct/map
	if data["ConnectedVia"] == "NotConnected" {
		return false
	}
	return true
}

func (redfishClient RedfishClient) InsertISO( image_url string) (bool) {
	fmt.Printf("Starting ISO attach\n")
	if redfishClient.GetVirtualMediaStatus() == true {
		fmt.Printf("Skipping Iso Insert. CD already Attached\n")
		return false
	} else {
		fmt.Printf("Attachig new ISO %s\n", image_url)
		endpoint := redfishClient.ManagerURL("VirtualMedia","CD", "Actions", "VirtualMedia.InsertMedia")
		header := redfishClient.Header
		body := `{"Image": "` + image_url +`"}`
		r, err := redfishClient.HttpClient.Post(endpoint, header, body)
		CheckErrorAndReturn(r,err)
		return true
	}

}


func(redfishClient RedfishClient) SetOneTimeBoot () (bool) {
		// Actions/Oem/EID_674_Manager.ImportSystemConfiguration
		fmt.Printf("Setting Onetime boot to VirtualMediaCDROM\n")
		endpoint := redfishClient.ManagerURL("Actions","Oem", "EID_674_Manager.ImportSystemConfiguration")
		header := redfishClient.Header
		body := `{
			"ShareParameters": {
				"Target": "ALL"
			},
			"ImportBuffer": "<SystemConfiguration><Component FQDD=\"iDRAC.Embedded.1\"><Attribute Name=\"ServerBoot.1#BootOnce\">Enabled</Attribute><Attribute Name=\"ServerBoot.1#FirstBootDevice\">VCD-DVD</Attribute></Component></SystemConfiguration>"
		}`
		//fmt.Printf("Body is %s \n", body)
		r, err := redfishClient.HttpClient.Post(endpoint, header, body)
		CheckErrorAndReturn(r,err)
		return true

}

func(redfishClient RedfishClient) Reboot () (bool) {
	///Systems/System.Embedded.1/Actions/ComputerSystem.Reset
	fmt.Printf("Starting OS installation. Rebooting the node\n")
	endpoint := redfishClient.SystemURL("Actions","ComputerSystem.Reset")
	header := redfishClient.Header
	body := `{"ResetType" : "ForceRestart" }`
	//fmt.Printf("Body is %s \n", body)
	r, err := redfishClient.HttpClient.Post(endpoint, header, body)
	CheckErrorAndReturn(r,err)
	return true
}


func (redfishClient RedfishClient) EjectISO() (bool) {
	fmt.Printf("Starting ISO Eject\n")
	if redfishClient.GetVirtualMediaStatus() == false {
		fmt.Printf("No CD to eject\n")
	} else {
		fmt.Printf("Ejecting existing CD\n")
		endpoint := redfishClient.ManagerURL("VirtualMedia","CD", "Actions", "VirtualMedia.EjectMedia")
		header := redfishClient.Header
		body := `{}`
		r, err := redfishClient.HttpClient.Post(endpoint, header, body)
		CheckErrorAndReturn(r,err)
	}
	return true	
}

func (redfishClient RedfishClient)  GetUniqueNodeId(hostname string) (string) {

	h := md5.New()
    h.Write([]byte(strings.ToLower(hostname)))
    return hex.EncodeToString(h.Sum(nil))
}

func CheckErrorAndReturn(res *req.Resp, err error) (*req.Resp) {

	//fmt.Println(res)
	if err != nil {
		//log.Fatal(err)
		fmt.Println(err)
	}

	return res
}


func (redfishClient RedfishClient) Get(client RedfishClient, url string) (*req.Resp) {

	resp, _ := client.HttpClient.Get(url)

	return resp
}