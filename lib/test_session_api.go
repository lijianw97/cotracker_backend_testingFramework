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

type SessionGeneric struct {
	SessionID 	int `json:"sessionID"`
	IsAndroid 	bool `json:"isAndroid"`
	DeviceID  	string `json:"deviceID"`
	Alias 	  	string `json:"alias"`
	StartTime	string `json:"startTime"`
	EndTime	string `json:"endTime"`
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
		fmt.Println(err.Error())
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
		}
		fmt.Println(sessionid)
		hasSession =  true;
		fmt.Println("session unavailable")
	}
	rows.Close()
	if hasSession == false{
		fmt.Println("session available")
		q := `
		insert into Test_Sessions (sessionID) values (?);
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
		_, err = db.Exec(q2, req.IsAndroid, req.DeviceID,req.SessionID, time.Now().UTC())
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
func JoinSession(w http.ResponseWriter, r *http.Request){
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
	var req SessionDevice
	err = json.Unmarshal(body, &req)
	if err != nil{
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}
	row, err := db.Query("SELECT isActive from Test_Sessions WHERE sessionID=?", req.SessionID)
	defer row.Close()
	if err != nil{
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	if !row.Next(){
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid SessionID="+strconv.Itoa(req.SessionID)))
		return
	}
	var isActive bool
	row.Scan(&isActive)
	if !isActive{
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Inactive SessionID="+strconv.Itoa(req.SessionID)))
		return
	}
	row2, err := db.Query("SELECT * from Test_hasDevice WHERE isAndroid=? AND deviceID=? AND sessionID=? LIMIT 1", req.IsAndroid, req.DeviceID, req.SessionID)
	defer row2.Close()
	if err != nil{
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	if row2.Next(){
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Device already joined"))
		return
	}
	stmt, err := db.Prepare("INSERT INTO Test_hasDevice (isAndroid, deviceID, sessionID, startTime, alias) VALUES (?,?,?,?,?)")
	defer stmt.Close()
	if err != nil{
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	_, err = stmt.Exec(req.IsAndroid, req.DeviceID, req.SessionID, req.StartTime, req.Alias)
	if err != nil{
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	w.WriteHeader(http.StatusOK)
	//} else {
	//	fmt.Println("sucess connected to database")
	//	q := `select max(sessionID) + 1 from Test_Sessions`
	//	rows, err := db.Query(q)
	//	if (err != nil){
	//		log.Fatal(err)
	//	}
	//	defer rows.Close()
	//	var sessionID int
	//	rows.Next()
	//	rows.Scan(&sessionID)
	//	fmt.Println("sesson id is ?", sessionID)
	//	t:=strconv.Itoa(sessionID)
	//	 // var a = "sesson id is " + t
	//	w.Write([]byte(t))
	//}
}

