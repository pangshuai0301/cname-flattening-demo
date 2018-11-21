package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
        "net/url"
	"os"
	"strings"
	"errors"
)

type Domain struct {
	Dnspod_ns	[]string	`json:"dnspod_ns"`
	Ext_status	string		`json:"ext_status"`
	Grade		string		`json:"grade"`
        Id		int		`json:"id"`
	Min_ttl		int		`json:"min_ttl"`
	Name		string		`json:"name"`
	Owner		string		`json:"owner"`
	Punycode	string		`json:"punycode"`
	Status		string		`json:"status"`
	Ttl		int		`json:"ttl"`
}

type Info struct {
	Record_total	string		`json:"record_total"`
	Records_sum	string		`json:"records_sum"`
	Sum_domains	string		`json:"sub_domains"`
}

type Record struct {
	Enable		string		`json:"enabled"`
	Hold		string		`json:"hold"`
	Id		string		`json:"id"`
	Line		string		`json:"line"`
	Line_id		string		`json:"line_id"`
	Monitor_status	string		`json:"monitor_status"`
	Mx		string		`json:"mx"`
	Name		string		`json:"name"`
	Remark		string		`json:"remake"`
	Status		string		`json:"status"`
	Ttl		string		`json:"ttl"`
	Type		string		`json:"type"`
	Updated_on	string		`json:"updated_on"`
	Use_aqb		string		`json:"use_aqb"`
	Value		string		`json:"value"`
	Weight		string		`json:"weight"`
}

type Status struct {
	Code		string		`json:"code"`
	Create_at	string		`json:"create_at"`
	Message		string		`json:"message"`
}

type R_AddRecord struct {
	Id		string		`json:"id"`
	Name		string		`json:"name"`
	Status		string		`json:"status"`
}

type R_Status_Add struct {
	Status		Status		`json:"status"`
	Record		R_AddRecord	`json:"record"`
}

type R_Status_Remove struct {
	Status		Status		`json:"status"`
}

type DomainList struct {
	Domain		Domain		`json:"domain"`
	Info		Info		`json:"info"`
	Records		[]Record	`json:"records"`
	Status		Status		`json:"status"`
}


func main() {

	err := CheckRecord()
	if err != nil {
		fmt.Println("something err, ", err)
	}

}

func CheckRecord() error {

	WebDomainRecord, err := GetDomainRecords()
	LBDomainRecord, err1 := net.LookupHost("aws-devops-demo-8082-8081-579254150.cn-northwest-1.elb.amazonaws.com.cn")

	if err != nil {
		fmt.Println("Err WebDomainRecord: ", err.Error())
		os.Exit(-1)
	} else if err1 != nil {
		fmt.Println("Err LBDomainRecord: ", err1.Error())
		os.Exit(-2)
	}

        var DelArray []string
        var AddArray []string

        for ip, id := range WebDomainRecord {
                FlagExist := 0
                for _, n := range LBDomainRecord {
                        if n == ip {
			        FlagExist = 1
				break
                        }
		}
                if FlagExist == 0 {
                	err = RemoveDomainRecord(id) // record id
                	if err != nil {
                	        fmt.Println(err, "ip: ",ip)
                	        continue
                	}
                	fmt.Println("del record success.../ value: ", ip)
                        DelArray = append(DelArray, ip)
		}
        }

	for _, n := range LBDomainRecord {
		FlagNeedAdd := 1
        	for ip, _ := range WebDomainRecord {
			if n == ip {
			        FlagNeedAdd = 0
				break
			}
		}
		if FlagNeedAdd == 1 {
			err = CreateDomainRecord("@", "A", "0", n) // subname, type, line, value
			if err != nil {
				fmt.Println(err)
				continue
			}
			fmt.Println("Add record success.../ value: ", n)
			AddArray = append(AddArray, n)
        	}
	}

	fmt.Printf("Del %d record\n", len(DelArray))
	fmt.Printf("Add %d record\n", len(AddArray))

	return nil
}

func GetDomainRecords() (map[string]string, error) {

    Ip_Id := make(map[string]string)

    values := url.Values{}

    url := "https://dnsapi.cn/Record.List"

    resp, err := PostData(url, values)
    if err != nil {
        fmt.Println("Post Data Error...")
        return Ip_Id, err
    }

    var d DomainList
    err = json.Unmarshal([]byte(resp), &d)
    if err != nil {
        fmt.Println("Paser json Error...")
        return Ip_Id, err
    }

    if string(d.Status.Code) == "1" {
        records := d.Records
        for _, record := range records {
            enable := record.Enable
            if enable != "1" {
                continue
            }
            record_type := record.Type
            if record_type != "A" {
                continue
            }
            name := record.Name
            if name != "@" {
                continue
            }
            Value := record.Value
            Id := record.Id
            Ip_Id[Value] = Id
        }
    }

    return Ip_Id, nil
}

func CreateDomainRecord(Subdomain string, RecordType string, RecordLineId string, Value string) error {

    values := url.Values{}
    values.Add("sub_domain", Subdomain)
    values.Add("record_type", RecordType)
    values.Add("record_line_id", RecordLineId)
    values.Add("value", Value)
    url := "https://dnsapi.cn/Record.Create"

    resp, err := PostData(url, values)
    if err != nil {
        fmt.Println("Post Data Error...")
        return err
    }

    var r R_Status_Add
    err = json.Unmarshal([]byte(resp), &r)
    if err != nil {
        fmt.Println("Paser json Error...")
        return err
    }

    //fmt.Printf("%+v", r)

    if r.Status.Code != "1" {
        Errlog := fmt.Sprintf("CreateDomainRecord Err %s, code: %s", Value, r.Status.Code)
        return errors.New(Errlog)
    }

    return nil

}

func RemoveDomainRecord(RecordId string) error {

    values := url.Values{}
    values.Add("record_id", RecordId)
    url := "https://dnsapi.cn/Record.Remove"

    resp, err := PostData(url, values)
    if err != nil {
        fmt.Println("Post Data Error...")
        return err
    }

    var r R_Status_Remove
    err = json.Unmarshal([]byte(resp), &r)
    if err != nil {
        fmt.Println("Paser json Error...")
        return err
    }

    if r.Status.Code != "1" {
        Errlog := fmt.Sprintf("RemoveDomainRecord, code: %s", r.Status.Code)
        return errors.New(Errlog)
    }

    //fmt.Printf("%+v", r)

    return nil

}

func PostData(url string, content url.Values) (string, error) {

    LoginToken := os.Getenv("LOGINTOKEN")
    if LoginToken == "" {
        fmt.Println("Get LOGINTOKEN Failed...")
        os.Exit(-1)
    }
    client := &http.Client{}
    content.Add("login_token", LoginToken)
    content.Add("format", "json")
    content.Add("domain_id", "69818749")
    //content.Add("domain", "pstrive.org")

    req, _ := http.NewRequest("POST", url, strings.NewReader(content.Encode()))

    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    req.Header.Set("User-Agent", fmt.Sprintf("GoResolv/0.1 (%s)", ""))

    response, err := client.Do(req)

    if err != nil {
        fmt.Println("Post failed...")
        fmt.Println(err)
        return "", err
    }

    defer response.Body.Close()
    resp, _ := ioutil.ReadAll(response.Body)

    return string(resp), nil
}
