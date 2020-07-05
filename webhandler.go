package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	functions "./lib"
	_ "github.com/go-sql-driver/mysql"
)


const url string = "root:wang7203311@tcp(database-2.c1gw860hlwji.us-east-2.rds.amazonaws.com:3306)/LocationTable"

type TekStruct []struct {
	Timestemp int64  `json:"i" validate:"required"`
	Tek       string `json:"tek" validate:"required"`
}

type TekStructDevice []struct {
	Timestemp int64  `json:"i" validate:"required"`
	Tek       string `json:"tek" validate:"required"`
	IsAndroid	bool
	DeviceID	string
}

type SingleTek struct {
	Timestemp int64  `json:"i" validate:"required"`
	Tek       string `json:"tek" validate:"required"` }

type DeviceData struct {
	Userid           string `json:"userid"`
	MAC_Address      string `json:"MAC_Address"`
	TEK              string `json:"TEK"`
	RecvRPI          string `json:"recvRPI"`
	ExposureDuration int64  `json:"exposureDuration"`
	EndContact_ts    string `json:"EndContact_ts"`
	CreateTime       string `json:"createTime"`
	UpdateTime       string `json:"updateTime"`
	Test_Devicescol  string `json:"Test_Devicescol"`
}
type ExposureData struct {
	SessionID		int
	IsAndroid		bool
	DeviceID		string
	RPI				string
	StartTime		string
	Duration		int
	Source 			string
	PeripheralUuid 	string
}
type TekRpiData struct{
	TEK 			string
	TEKStartTime 	string
	DeviceID		string
	IsAndroid		bool
	RPI				string
	RPIStartTime	string
	Event 			string
}
func gettek(w http.ResponseWriter, r *http.Request) {
	fmt.Println("connected!!!!")
	body, _ := ioutil.ReadAll(r.Body)
	var requestdata TekStruct
	err := json.Unmarshal(body, &requestdata)
	if err != nil {
		log.Printf("error here")
	}
	db, err := sql.Open("mysql", url)
	defer db.Close()
	var response []byte
	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Println("sucess connected gettek")
		result, _ := db.Query("SELECT * FROM TekTable")
		var responsedata []SingleTek
		var tek1 string
		var tstamp1 int64
		for result.Next() {
			result.Scan(&tstamp1, &tek1)
			log.Println(tstamp1, tek1)
			temptek := SingleTek{tstamp1, tek1}
			responsedata = append(responsedata, temptek)
		}
		response, err = json.Marshal(responsedata)
		if err != nil {
			fmt.Println(err.Error())
		}
		fmt.Printf("response : %s\n", response)
	}
	//fmt.Printf("%s\n", requestdata[0].Tek)
	w.Write([]byte(response))
}

func posttek(w http.ResponseWriter, r *http.Request) {
	fmt.Println("connected!!!!")
	body, _ := ioutil.ReadAll(r.Body)
	// fmt.Println(body)
	var requestdata TekStruct
	err := json.Unmarshal(body, &requestdata)
	if err != nil {
		log.Printf("error here")
	}
	db, err := sql.Open("mysql", url)
	defer db.Close()
	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Println("sucess connected posttek")
		for _, v := range requestdata {
			insert, _ := db.Query("INSERT IGNORE INTO TekTable VALUES (?,?)",
				v.Timestemp, v.Tek)
			defer insert.Close()
		}
	}
	//fmt.Printf("log time = %d and tek = %s\n", requestdata[0].Timestemp, requestdata[0].Tek)
	w.Write([]byte("This is response"))
	fmt.Println("connected!!!!")
}

func addDevice(w http.ResponseWriter, r *http.Request) {
	fmt.Println("start AddDevice")
	body, _ := ioutil.ReadAll(r.Body)
	fmt.Println(body)
	var requestdata []DeviceData
	err := json.Unmarshal(body, &requestdata)
	if err != nil {
		log.Println(err)
	}
	db, err := sql.Open("mysql", url)
	defer fmt.Println("db closed")
	defer db.Close()
	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Println("sucess connected to database")
		for _, v := range requestdata {
			insert, _ := db.Query("INSERT INTO _deprecated_Test_Devices( userid, MAC_Address, TEK, recvRPI, exposureDuration, EndContact_ts, createTime, updateTime, Test_Devicescol) VALUES (?,?,?,?,?,?,?,?,?)",
				v.Userid, v.MAC_Address, v.TEK, v.RecvRPI, v.ExposureDuration, v.EndContact_ts, v.CreateTime, v.UpdateTime, v.Test_Devicescol)
			defer fmt.Println("insert closed")
			defer insert.Close()
		}
	}
	fmt.Printf("log %+v\n", requestdata)
	w.Write([]byte("device inserted"))
	fmt.Println("end AddDevice")
}

// func createSession(w http.ResponseWriter, r *http.Request) {
// 	fmt.Println("start createSession")
// 	body, _ := ioutil.ReadAll(r.Body)
// 	fmt.Println(body)
// 	var requestdata []DeviceData
// 	err := json.Unmarshal(body, &requestdata)
// 	if err != nil {
// 		log.Println(err)
// 	}
// 	db, err := sql.Open("mysql", url)
// 	defer fmt.Println("db closed")
// 	defer db.Close()
// 	if err != nil {
// 		fmt.Println(err.Error())
// 	} else {
// 		fmt.Println("sucess connected to database")
// 		for _, v := range requestdata {
// 			insert, _ := db.Query("INSERT INTO _deprecated_Test_Devices( userid, MAC_Address, TEK, recvRPI, exposureDuration, EndContact_ts, createTime, updateTime, Test_Devicescol) VALUES (?,?,?,?,?,?,?,?,?)",
// 				v.Userid, v.MAC_Address, v.TEK, v.RecvRPI, v.ExposureDuration, v.EndContact_ts, v.CreateTime, v.UpdateTime, v.Test_Devicescol)
// 			defer fmt.Println("insert closed")
// 			defer insert.Close()
// 		}
// 	}
// 	fmt.Printf("log %+v\n", requestdata)
// 	w.Write([]byte("device inserted"))
// 	fmt.Println("end AddDevice")
// }
func insertExposure(w http.ResponseWriter, r *http.Request){
	body, _ := ioutil.ReadAll(r.Body)
	fmt.Println(body)
	var requestdata []ExposureData
	var db *sql.DB
	var err error
	err = json.Unmarshal(body, &requestdata)
	if err != nil {
		log.Println(err)
	}
	db, err = sql.Open("mysql", url)

	if err != nil {
		fmt.Println(err.Error())
		return
	}
	defer db.Close()
	fmt.Println("sucess connected to database")
	stmt, err := db.Prepare("INSERT INTO Test_Exposures (sessionID, isAndroid, deviceID, RPI, startTime, duration, source, peripheralUuid) VALUES (?,?,?,?,?,?,?,?)")
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	defer stmt.Close()
	for _, v := range requestdata {
		_, err := stmt.Exec(v.SessionID, v.IsAndroid, v.DeviceID, v.RPI, v.StartTime, v.Duration, v.Source, v.PeripheralUuid) // @TODO: change to Exec
		if err != nil {
			fmt.Println(err.Error())
		}
		defer fmt.Println("insert closed")
	}
	fmt.Printf("log %+v\n", requestdata)
	w.Write([]byte("device inserted"))
}

func posttekWithDevice(w http.ResponseWriter, r *http.Request) {
	fmt.Println("connected!!!!")
	body, _ := ioutil.ReadAll(r.Body)
	// fmt.Println(body)
	var requestdata TekStructDevice
	err := json.Unmarshal(body, &requestdata)
	if err != nil {
		log.Printf("error here")
	}
	db, err := sql.Open("mysql", url)
	defer db.Close()
	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Println("sucess connected posttekWithDevice")
		for _, v := range requestdata {
			insert, _ := db.Query("INSERT INTO Test_TEK (TEK, startTime, deviceID, isAndroid) VALUES (?,?,?,?)",
				v.Tek, v.Timestemp, v.DeviceID, v.IsAndroid)
			defer insert.Close()
		}
	}
	//fmt.Printf("log time = %d and tek = %s\n", requestdata[0].Timestemp, requestdata[0].Tek)
	w.Write([]byte("This is response"))
	fmt.Println("connected!!!!")
}

func updateTekRpi(w http.ResponseWriter, r *http.Request) {
	fmt.Println("connected!!!!")
	body, _ := ioutil.ReadAll(r.Body)
	var requestdata TekRpiData
	err := json.Unmarshal(body, &requestdata)
	if err != nil {
		log.Printf("error here")
		return
	}
	db, err := sql.Open("mysql", url)
	defer db.Close()
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println("sucess connected PostRpiWithTek")
	row, err := db.Query("SELECT TEK, startTime, deviceID, isAndroid FROM Test_TEK WHERE TEK = ? AND deviceID = ? AND isAndroid = ? LIMIT 1",
		requestdata.TEK, requestdata.DeviceID, requestdata.IsAndroid)
	if err != nil{
		fmt.Println(err.Error())
		return
	}
	defer row.Close()
	if !row.Next() || requestdata.TEK == ""{ // insert into Test_TEK if entry/TEK missing
		stmtTek, err := db.Prepare("INSERT INTO Test_TEK (TEK, startTime, deviceID, isAndroid) VALUES (?,?,?,?)")
		if err != nil{
			fmt.Println(err.Error())
			return
		}
		defer stmtTek.Close()
		_, err = stmtTek.Exec(requestdata.TEK, requestdata.TEKStartTime, requestdata.DeviceID, requestdata.IsAndroid)
		if err != nil{
			fmt.Println(err.Error())
			return
		}
	}
	stmtRpi, err := db.Prepare("INSERT INTO Test_RPI (TEK, TEKStartTime, deviceID, isAndroid, RPI, RPIStartTime, event) VALUES (?, ?, ?, ?, ?, ?, ?)")
	if err != nil{
		fmt.Println(err.Error())
		return
	}
	defer stmtRpi.Close()
	_, err = stmtRpi.Exec(requestdata.TEK, requestdata.TEKStartTime, requestdata.DeviceID, requestdata.IsAndroid, requestdata.RPI, requestdata.RPIStartTime, requestdata.Event)
	if err != nil{
		fmt.Println(err.Error())
		return
	}
	w.Write([]byte("Success: tek=" + requestdata.TEK + "rpi=" + requestdata.RPI))
	fmt.Println("connected!!!!")
}

func main() {
	fmt.Println("listen")
	http.HandleFunc("/PostTek", posttek)
	http.HandleFunc("/GetTek", gettek)
	http.HandleFunc("/AddDevice", addDevice)
	http.HandleFunc("/AddExposure", insertExposure)
	http.HandleFunc("/PostTekWithDevice", posttekWithDevice)
	http.HandleFunc("/PostRpiWithTek", updateTekRpi)
	fmt.Println(functions.Test())
	log.Fatal(http.ListenAndServe(":8003", nil))
}
