package functions 

import (
 	"database/sql"
 	"encoding/json"
    "fmt"
 	"io/ioutil"
 	"log"
	"time"
 	"net/http"
	"strconv"
	_ "github.com/go-sql-driver/mysql"
)
const url string = "root:wang7203311@tcp(database-2.c1gw860hlwji.us-east-2.rds.amazonaws.com:3306)/LocationTable"

type SessionGeneric struct {
	SessionID 	int `json:"sessionID"`
	IsAndroid 	bool `json:"isAndroid"`
	DeviceID  	string `json:"deviceID"`
	Alias 	  	string `json:"alias"`
	AdditionalDetail		string `json:"additionalDetail"`
	StartTime	string `json:"startTime"`
	EndTime	string `json:"endTime"`
}


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

type SessionDevice struct {
	SessionID 	int
	IsAndroid 	bool
	DeviceID  	string
	Alias 	  	string
	StartTime	string
}

type SessionList []struct {
	SessionID 	int
	IsAndroid 	bool
}

func Test() string{
	return "testing success"
}

func GetSessionID(w http.ResponseWriter, r *http.Request) {
	fmt.Println("querying for session")
	db, err := sql.Open("mysql", url)
	defer fmt.Println("db closed")
	defer db.Close()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	} else {
		fmt.Println("sucess connected to database")
		q := `select max(sessionID) + 1 from Test_Sessions`
		rows, err := db.Query(q)
		if (err != nil){
			log.Fatal(err)
		}
		defer rows.Close()
		var sessionID int
		rows.Next()
		rows.Scan(&sessionID)
		fmt.Println("sesson id is ?", sessionID)
		t:=strconv.Itoa(sessionID)
		w.Write([]byte(t))
	}
}

// insert Test_Sessions table given (LIST) sessions
// eg: curl -X post -i http://ec2-18-191-37-235.us-east-2.compute.amazonaws.com:8003/CreateSession --data '[{"sessionID": -2,"isActive": false}, {"sessionID": -3,"isActive": false}]'
// CreateSession:
// request json field: sessionID, isAndroid, deviceID
// respones json field: null; 
// on success: status code = 200 http.StatusOK and return message
// on failed: status code = 500 http.StatusInternalServerError and return message
func CreateSession(w http.ResponseWriter, r *http.Request){
	fmt.Println("querying for session")
	db, err := sql.Open("mysql", url)
	defer fmt.Println("db closed")
	defer db.Close()
	if err != nil {
		fmt.Println(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	body, _ := ioutil.ReadAll(r.Body)
	var req SessionGeneric
	err = json.Unmarshal(body, &req)
	fmt.Printf("%+v\n",req) // Print with Variable Name
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}
	//deal with device info first 
	stmt, err := db.Prepare("INSERT IGNORE INTO Test_Devices(isAndroid, ID) VALUES (?,?)")
	defer stmt.Close()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	_, err = stmt.Exec(req.IsAndroid, req.DeviceID)
	if err!= nil{
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	stmt.Close()
	q := `
	select sessionID from Test_Sessions where sessionID = ? ;
	`
	rows, err := db.Query(q , req.SessionID)
	hasSession := false
	defer rows.Close()
	for rows.Next(){
		var sessionid int
		if err := rows.Scan(&sessionid); err != nil {
			hasSession = false
			log.Fatal(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		fmt.Println(sessionid)
		hasSession =  true;
		fmt.Println("session unavailable")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("session unavailable"))
		return
	}
	rows.Close()
	if hasSession == false{
		fmt.Println("session available")
		q := `
		insert into Test_Sessions (sessionID, isActive) values (?, true);
		`
		q2:=`
		insert into Test_hasDevice (isAndroid, deviceID, sessionID, startTime) values (?,?,?,?);
		`
		_, err = db.Exec(q,req.SessionID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		_, err = db.Exec(q2, req.IsAndroid, req.DeviceID,req.SessionID, time.Now())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		w.Write([]byte("session created and joined!"))

	}
}

// insert Test_hasDevice table given (SINGLE) correct sessionID
// eg: curl -X post -i http://ec2-18-191-37-235.us-east-2.compute.amazonaws.com:8003/JoinSession --data '{"sessionID": 0, "deviceID": "c72972f5-301d-43d1-b3e6-b3b58ea84386", "isAndroid":false, "startTime": "2020-05-07 23:39:18", "alias": "my iphone"}'

// JoinSession:
// request json field: sessionID, isAndroid, deviceID
// respones json field: null; 
// on success: status code = 200 http.StatusOK and return message
// on failed: status code = 500 http.StatusInternalServerError and return message
func JoinSession(w http.ResponseWriter, r *http.Request){
	fmt.Println("querying for target session")
	db, err := sql.Open("mysql", url)
	defer fmt.Println("db closed")
	defer db.Close()
	if err != nil {
		fmt.Println(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	body, _ := ioutil.ReadAll(r.Body)
	var req SessionGeneric
	err = json.Unmarshal(body, &req)
	fmt.Printf("%+v\n",req) // Print with Variable Name
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}
	// deal with device info first
	stmt, err := db.Prepare("INSERT IGNORE INTO Test_Devices(isAndroid, ID) VALUES (?,?)")
	defer stmt.Close()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	_, err = stmt.Exec(req.IsAndroid, req.DeviceID)
	if err!= nil{
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	stmt.Close()
	q := `
	select sessionID from Test_Sessions where sessionID = ? and isActive = 1;
	`
	rows, err := db.Query(q , req.SessionID)
	hasSession := false
	defer rows.Close()
	for rows.Next(){
		var sessionid int
		if err := rows.Scan(&sessionid); err != nil {
			fmt.Println("session is not active")
			hasSession = false
			log.Fatal(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("target session is not active"))
			return
		}
		fmt.Println(sessionid)
		hasSession =  true;
		fmt.Println("active session found")
	}
	if hasSession == false{
		fmt.Println("no available session")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("target session is not active"))
		return
	}
	rows.Close()
	if hasSession == true{
		fmt.Println("session available")
		q2:=`
		insert ignore into Test_hasDevice (isAndroid, deviceID, sessionID, startTime) values (?,?,?,?);
		`
		_, err = db.Exec(q2, req.IsAndroid, req.DeviceID,req.SessionID, time.Now())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		w.Write([]byte("session joined!"))

	}
}

/**
EndSession: 
req json field: isAndroid, deviceID, sessionID, additionalDetail
res json field: null
Add an endTime to device. 
Check if there is any other device with the same 
*/
func EndSession(w http.ResponseWriter, r *http.Request){
	fmt.Println("ending session")
	db, err := sql.Open("mysql", url)
	defer fmt.Println("db closed")
	defer db.Close()
	if err != nil {
		fmt.Println(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	body, _ := ioutil.ReadAll(r.Body)
	var req SessionGeneric
	err = json.Unmarshal(body, &req)
	fmt.Printf("%+v\n",req) // Print with Variable Name
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}
	q := `
	update Test_hasDevice set endTime = ? where isAndroid = ? and deviceID = ? and sessionID = ? ; 
	`
	_, err = db.Exec(q, time.Now(), req.IsAndroid, req.DeviceID, req.SessionID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	// now check if I should set a session as inactive 
	// query for all devices given a sessionID which is unique across time. 
	// if any of them has endTime as null, this session is active
	q1 := `
	select count(*) from Test_hasDevice where sessionID = ? and endTime is null;
	`
	rows, err1 := db.Query(q1, req.SessionID)
	if err1 != nil{
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err1.Error()))
		return
	}
	defer rows.Close()
	var count int
	rows.Next()
	rows.Scan(&count)
	fmt.Printf("%d device is still active in the session",count)
	if count != 0{
		w.Write([]byte(fmt.Sprintf("Session ended but %d devices active", count)))
		return
	} else{
		fmt.Println("no active device in the target session")
		q2 := `
		update Test_Sessions set isActive = 0 where sessionID = ?;
		`
		_,err = db.Exec(q2, req.SessionID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		w.Write([]byte("Session ended and is inactive"))
	}
}

/**
SessionReport: 
parameter: should be taken from url parameters in a GET method. 
Check if there is any other device with the same 
*/
func SessionReport(w http.ResponseWriter, r *http.Request){
	var (
		sessionIDs []string
		deviceIndexes []string
	)
	queryMap:= r.URL.Query()
	sessionIDs = queryMap["sessionID"]
	deviceIndexes = queryMap["deviceIndex"]
	fmt.Println(queryMap)
	fmt.Println(len(sessionIDs))
	fmt.Println(len(deviceIndexes))
	if len(sessionIDs) == 0{
		resp := `
No sessionID. Please consider appending "?sessionID=1&deviceIndex=1" at the end of the url for specific session or specific device. 
Maybe I should list out all sessionIDs, 
and show how many sessions are ongoing and how many are inactive. 
Maybe also show stats about when those session begin and end.
As well as number of ongoing devices and stuff. 
		`
		resp += _reportSessionWithoutID()
		w.Write([]byte(resp))
		return 
	} else if len(deviceIndexes) == 0{
		// resp := `
		// Summary of the selected session. 
		// No device specific details
		// `
		resp := ``
		sessionID, err :=  strconv.Atoi(sessionIDs[0])
		if err != nil{
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		resp += _reportSessionWithSessionID(sessionID)
		w.Write([]byte(resp))
		return 
	} else{
		resp := ``
		sessionID, err :=  strconv.Atoi(sessionIDs[0])
		if err != nil{
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		deviceIndex, err :=  strconv.Atoi(deviceIndexes[0])
		if err != nil{
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		resp += _reportSessionWithBothID(sessionID, deviceIndex, w, r)
		w.Write([]byte(resp))
		return
	}

}

/*
session report with both ids present: 
query for session status 
query for device status; print out total number of device in the session. number of active and inactive 
*/
func _reportSessionWithBothID(sessionID ,deviceIndex int, w http.ResponseWriter, r *http.Request) string {
	var (
		err		error
		db		*sql.DB
		rows	*sql.Rows
		// rows2	*sql.Rows
		q1		string
		content []string
		sessionStatus string
		deviceCount string
		deviceStatus []string // total 
		// i			int // iterator index
	)
	db, err = sql.Open("mysql", url)
	defer fmt.Println("db closed")
	defer db.Close()
	if err != nil {
		fmt.Println(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return ""
	}
	q1 = `
		select isActive from Test_Sessions where sessionID=?;
	`
	rows,err = db.Query(q1, sessionID)
	if err != nil{
		fmt.Println(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return ""
	}
	defer rows.Close()
	rows.Next()
	var isActive int
	rows.Scan(&isActive)
	rows.Close() // for reusing
	if isActive == 1{
		sessionStatus = "Active, expect incomplete data"
	} else{
		sessionStatus = "Inactive, data can be complete"
	}
	content = append(content, "Session Status",sessionStatus)

	deviceStatus = nil
	// get total devices 
	q1 = `
	select count(*) from Test_hasDevice where sessionID = ?;
	`
	rows, err = db.Query(q1,sessionID)
	if err!=nil{
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return  err.Error()
	}
	rows.Next()
	if err := rows.Scan(&deviceCount); err != nil{
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return  err.Error()
	}
	rows.Close()
	fmt.Println("device count" + deviceCount)
	deviceStatus = append(deviceStatus, deviceCount)

	// get inactive devices 
	q1 = `
	select count(*) from Test_hasDevice where sessionID = ? and endTime is not null;
	`
	rows, err = db.Query(q1,sessionID)
	if err!=nil{
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return  err.Error()
	}
	rows.Next()
	if err := rows.Scan(&deviceCount); err != nil{
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return  err.Error()
	}
	rows.Close()
	fmt.Println("device count" + deviceCount)
	deviceStatus = append(deviceStatus, deviceCount)

	// get acitve devices 
	q1 = `
	select count(*) from Test_hasDevice where sessionID = ? and endTime is null;
	`
	rows, err = db.Query(q1,sessionID)
	if err!=nil{
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return  err.Error()
	}
	rows.Next()
	if err := rows.Scan(&deviceCount); err != nil{
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return  err.Error()
	}
	rows.Close()
	fmt.Println("device count" + deviceCount)
	deviceStatus = append(deviceStatus, deviceCount)

	content = append(content, "Device Status", fmt.Sprintf("There are, in total, %s devices , %s are inactive and %s are active", deviceStatus[0],deviceStatus[1], deviceStatus[2],  ));
	deviceStatus = nil

	// get exposures 
	return _htmlify(content)
}

func _reportSessionWithoutID() string {
	return _htmlify([]string{"title", "content", "another", "content2"})
}

func _reportSessionWithSessionID(sessionID int ) string {
	return _htmlify([]string{"title", "content", "another", "content2"})
}

// input: array of string that follows
// []string{topic1, content1, topic2, content2,...}
// Function is intended to work with even numbered input string 
// where the former element in a pair is topic and latter is content for that topic
// output: html string for result rendering
// input array of string in a paired way 0,1 2,3 4,5
func _htmlify(content []string) string {
	pre:=`
<!DOCTYPE html>
<html>
<head>
<title>Test Session Report</title>
</head>
<body>
	`
	post:=`
</body>
</html>
	`
	var body string
	body = ""
	if len(content) % 2 != 0{
		body = "ERROR! Content is not even numbered"
	} else{
		for i:=0; i < len(content); i+=2{
			body +=fmt.Sprintf(`
			<h3>%s</h3>
			<p>%s</p>
			<hr>
			`, content[i], content[i+1])
		}
	}
	return pre + body + post
}

func _easyQuery(q string, args ...interface{}) string{
	return ""
}
