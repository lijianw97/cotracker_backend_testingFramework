package functions

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/goccy/go-graphviz"
)

const url string = "root:wang7203311@tcp(database-2.c1gw860hlwji.us-east-2.rds.amazonaws.com:3306)/LocationTable"
const queryRelativeURLwithDeviceIndexThenSessionID string = "SessionReport?deviceIndex=%s&sessionID=%s"

type SessionGeneric struct {
	SessionID        int    `json:"sessionID"`
	IsAndroid        bool   `json:"isAndroid"`
	DeviceID         string `json:"deviceID"`
	Alias            string `json:"alias"`
	AdditionalDetail string `json:"additionalDetail"`
	StartTime        string `json:"startTime"`
	EndTime          string `json:"endTime"`
	Message          string `json:"message"`
	DeviceIndex      string `json:"deviceIndex"`
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
	SessionID int
	IsAndroid bool
	DeviceID  string
	Alias     string
	StartTime string
}

type SessionList []struct {
	SessionID int
	IsAndroid bool
}

func Test() string {
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
		if err != nil {
			log.Fatal(err)
		}
		defer rows.Close()
		var sessionID int
		rows.Next()
		rows.Scan(&sessionID)
		fmt.Println("sesson id is ?", sessionID)
		t := strconv.Itoa(sessionID)
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
func CreateSession(w http.ResponseWriter, r *http.Request) {
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
	fmt.Printf("%+v\n", req) // Print with Variable Name
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
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	stmt.Close()
	q := `
	select sessionID from Test_Sessions where sessionID = ? ;
	`
	rows, err := db.Query(q, req.SessionID)
	hasSession := false
	defer rows.Close()
	for rows.Next() {
		var sessionid int
		if err := rows.Scan(&sessionid); err != nil {
			hasSession = false
			log.Fatal(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		fmt.Println(sessionid)
		hasSession = true
		fmt.Println("session unavailable")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("session unavailable"))
		return
	}
	rows.Close()
	if hasSession == false {
		fmt.Println("session available")
		q := `
		insert into Test_Sessions (sessionID, isActive) values (?, true);
		`
		q2 := `
		insert into Test_hasDevice (isAndroid, deviceID, sessionID, startTime) values (?,?,?,?);
		`
		_, err = db.Exec(q, req.SessionID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		tm := time.Now()
		_, err = db.Exec(q2, req.IsAndroid, req.DeviceID, req.SessionID, tm)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		// about the query below, if there are multiple open connections to the same session, get the first one always. When ending session, both connection will be closed
		q = `
		select deviceIndex from Test_hasDevice where isAndroid=? and deviceID =? and sessionID = ? and endTime is null order by deviceIndex asc;
		`
		rows, err = db.Query(q, req.IsAndroid, req.DeviceID, req.SessionID)
		if err != nil {
			fmt.Println(err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		rows.Next()
		var deviceIdx string
		if err = rows.Scan(&deviceIdx); err != nil {
			fmt.Println(err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		fmt.Println("Device Index is ", deviceIdx)
		rows.Close()
		var response SessionGeneric
		response.DeviceIndex = deviceIdx
		response.Message = "session created and joined!"
		js, errr := json.Marshal(response)
		fmt.Println("response" + string(js))
		if errr != nil {
			fmt.Println(err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		w.Write(js)

	}
}

// insert Test_hasDevice table given (SINGLE) correct sessionID
// eg: curl -X post -i http://ec2-18-191-37-235.us-east-2.compute.amazonaws.com:8003/JoinSession --data '{"sessionID": 0, "deviceID": "c72972f5-301d-43d1-b3e6-b3b58ea84386", "isAndroid":false, "startTime": "2020-05-07 23:39:18", "alias": "my iphone"}'

// JoinSession:
// request json field: sessionID, isAndroid, deviceID
// respones json field: null;
// on success: status code = 200 http.StatusOK and return message
// on failed: status code = 500 http.StatusInternalServerError and return message
func JoinSession(w http.ResponseWriter, r *http.Request) {
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
	fmt.Printf("%+v\n", req) // Print with Variable Name
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
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	stmt.Close()
	q := `
	select sessionID from Test_Sessions where sessionID = ? and isActive = 1;
	`
	rows, err := db.Query(q, req.SessionID)
	hasSession := false
	defer rows.Close()
	for rows.Next() {
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
		hasSession = true
		fmt.Println("active session found")
	}
	if hasSession == false {
		fmt.Println("no available session")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("target session is not active"))
		return
	}
	rows.Close()
	if hasSession == true {
		fmt.Println("session available")
		q2 := `
		insert ignore into Test_hasDevice (isAndroid, deviceID, sessionID, startTime) values (?,?,?,?);
		`
		_, err = db.Exec(q2, req.IsAndroid, req.DeviceID, req.SessionID, time.Now())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		// get deviceIndex again
		// about the query below, if there are multiple open connections to the same session, get the first one always. When ending session, both connection will be closed
		q := `
		select deviceIndex from Test_hasDevice where isAndroid=? and deviceID =? and sessionID = ? and endTime is null order by deviceIndex asc;
		`
		rows, err = db.Query(q, req.IsAndroid, req.DeviceID, req.SessionID)
		if err != nil {
			fmt.Println(err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		rows.Next()
		var deviceIdx string
		if err = rows.Scan(&deviceIdx); err != nil {
			fmt.Println(err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		fmt.Println("Device Index is ", deviceIdx)
		rows.Close()
		var response SessionGeneric
		response.DeviceIndex = deviceIdx
		response.Message = "Session Joined!"
		js, errr := json.Marshal(response)
		fmt.Println("response" + string(js))
		if errr != nil {
			fmt.Println(err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		w.Write(js)

	}
}

/**
EndSession:
req json field: isAndroid, deviceID, sessionID, additionalDetail
res json field: null
Add an endTime to device.
Check if there is any other device with the same
*/
func EndSession(w http.ResponseWriter, r *http.Request) {
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
	fmt.Printf("%+v\n", req) // Print with Variable Name
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
	if err1 != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err1.Error()))
		return
	}
	defer rows.Close()
	var count int
	rows.Next()
	rows.Scan(&count)
	fmt.Printf("%d device is still active in the session", count)
	if count != 0 {
		w.Write([]byte(fmt.Sprintf("Session ended but %d devices active", count)))
		return
	} else {
		fmt.Println("no active device in the target session")
		q2 := `
		update Test_Sessions set isActive = 0 where sessionID = ?;
		`
		_, err = db.Exec(q2, req.SessionID)
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
func SessionReport(w http.ResponseWriter, r *http.Request) {
	var (
		sessionIDs    []string
		deviceIndexes []string
		err           error
		db            *sql.DB
	)
	db, err = sql.Open("mysql", url)
	defer fmt.Println("db closed")
	defer db.Close()
	if err != nil {
		fmt.Println(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	queryMap := r.URL.Query()
	sessionIDs = queryMap["sessionID"]
	deviceIndexes = queryMap["deviceIndex"]
	fmt.Println(queryMap)
	fmt.Println(len(sessionIDs))
	fmt.Println(len(deviceIndexes))
	if len(sessionIDs) == 0 {
		resp := _reportSessionWithoutSessionID(db)
		w.Write([]byte(resp))
		return
	} else if len(deviceIndexes) == 0 {
		// resp := `
		// Summary of the selected session.
		// No device specific details
		// `
		resp := ``
		sessionID := sessionIDs[0]
		resp += _reportSessionWithSessionID(sessionID, db)
		w.Write([]byte(resp))
		return
	} else {
		resp := ``
		// sessionID, err := strconv.Atoi(sessionIDs[0])
		sessionID := sessionIDs[0]
		// deviceIndex, err := strconv.Atoi(deviceIndexes[0])
		deviceIndex := deviceIndexes[0]
		resp += _reportSessionWithBothID(sessionID, deviceIndex, w, r, db)
		w.Write([]byte(resp))
		return
	}

}

/*
session report with both ids present:
query for session status
query for device status; print out total number of device in the session. number of active and inactive
*/
func _reportSessionWithBothID(sessionID, deviceIndex string,
	w http.ResponseWriter, r *http.Request, db *sql.DB) string {
	var (
		q1            string
		content       []string
		sessionStatus string
		deviceCount   string
		deviceStatus  []string // total
	)
	q1 = ` select isActive from Test_Sessions where sessionID=?; `
	isActive := _dangerouslyQueryForOneNumber(db, q1, sessionID)

	if isActive == "1" {
		sessionStatus = "Active, expect incomplete data"
	} else {
		sessionStatus = "Inactive, data can be complete"
	}
	content = append(content, fmt.Sprintf("Session %s Status", sessionID), sessionStatus)

	// get device status
	deviceStatus = nil
	// get total devices
	q1 = ` select count(*) from Test_hasDevice where sessionID = ?; `
	deviceCount = _dangerouslyQueryForOneNumber(db, q1, sessionID)
	fmt.Println("device count" + deviceCount)
	deviceStatus = append(deviceStatus, deviceCount)
	// get inactive device counts
	q1 = ` select count(*) from Test_hasDevice where sessionID = ? and endTime is not null; `
	deviceCount = _dangerouslyQueryForOneNumber(db, q1, sessionID)
	fmt.Println("device count" + deviceCount)
	deviceStatus = append(deviceStatus, deviceCount)
	// get acitve devices counts
	q1 = ` select count(*) from Test_hasDevice where sessionID = ? and endTime is null; `
	deviceCount = _dangerouslyQueryForOneNumber(db, q1, sessionID)
	fmt.Println("device count" + deviceCount)
	deviceStatus = append(deviceStatus, deviceCount)
	deviceStatusInterface := make([]interface{}, 3)
	for i := 0; i < 3; i++ {
		deviceStatusInterface[i] = deviceStatus[i]
	}
	deviceStatusContent := fmt.Sprintf("There are, in total, %s devices , %s are inactive and %s are active", deviceStatusInterface...)
	deviceStatusContent += _h4("Participating Devices")
	// query for inactive devices first
	q := `
	select deviceIndex, isAndroid, deviceID from Test_hasDevice where sessionID = ? and endTime is not null;
	`
	var participatingDevices []string
	var (
		selfDeviceID, selfIsAndroid string
	)
	participatingDeviceContent_out := _dangerouslyQueryForNNumberForMultipleLines(
		3, db, q, sessionID)
	for _, v := range participatingDeviceContent_out {
		var (
			vvvv, vv, vvv string
		)
		vvvv = "Inactive"
		if v[1] == "1" {
			vv = "Android"
		} else {
			vv = "iOS"
		}
		if string(v[0]) == deviceIndex {
			vvv = _bold("SELF")
			selfDeviceID = v[2]
			selfIsAndroid = v[1]
		}
		participatingDevices = append(participatingDevices,
			fmt.Sprintf("%s device index %s, %s, %s %s", vvvv,
				_a(fmt.Sprintf("SessionReport?deviceIndex=%s&sessionID=%s", v[0], sessionID), v[0]),
				vv, v[2], vvv))
	}
	deviceStatusContent += _ul(participatingDevices...)

	// query for active devices
	participatingDevices = nil
	q = `
	select deviceIndex, isAndroid, deviceID from Test_hasDevice where sessionID = ? and endTime is null;
	`
	participatingDeviceContent_out = _dangerouslyQueryForNNumberForMultipleLines(
		3, db, q, sessionID)

	for _, v := range participatingDeviceContent_out {
		var (
			vvvv, vv, vvv string
		)
		vvvv = "Active"
		if v[1] == "1" {
			vv = "Android"
		} else {
			vv = "iOS"
		}
		if string(v[0]) == deviceIndex {
			vvv = _bold("SELF")
			selfDeviceID = v[2]
			selfIsAndroid = v[1]
		}
		participatingDevices = append(participatingDevices,
			fmt.Sprintf("%s device index %s, %s, %s %s", vvvv,
				_a(fmt.Sprintf("SessionReport?deviceIndex=%s&sessionID=%s", v[0], sessionID), v[0]),
				vv, v[2], vvv))
	}
	deviceStatusContent += _ul(participatingDevices...)
	content = append(content, fmt.Sprintf("Device %s Status", deviceIndex), deviceStatusContent)
	// get a graph
	content = append(content, "Graph", _generateEncounterGraph(db, sessionID, deviceIndex))

	// get exposures
	q = `
	select c.duration, d.deviceIndex, e.RSSI, d.isAndroid, d.deviceID, 
	e.RPI
 from (select a.duration, b.isAndroid, b.deviceID, b.sessionID, b.RPI,
 	a.deviceID as thisDeviceID, a.isAndroid as thisIsAndroid
  from (select * from Test_Exposures where isAndroid = ? and deviceID = ? and sessionID = ?) a 
  inner join Test_RPI b on a.sessionID = b.sessionID and a.RPI = b.RPI )  c inner join 
 Test_hasDevice d on c.sessionID = d.sessionID and 
 c.isAndroid = d.isAndroid and c.deviceID = d.deviceID inner join (select 
 	RPI, isAndroid, deviceID, sessionID, sum(rssi)/count(*) as 
 	RSSI from Test_Exposures_Rssi where RSSI <> 127 group by isAndroid, 
 	deviceID, RPI, sessionID) e on c.sessionid = e.sessionid 
 and c.RPI = e.RPI and c.thisIsAndroid = e.isAndroid and c.thisDeviceID = e.deviceID;
	 `
	exposureQueryResults := _dangerouslyQueryForNNumberForMultipleLines(6, db, q, selfIsAndroid, selfDeviceID, sessionID)
	// I get duration, deviceIndex (other), RSSI (avg), isAndroid, deviceID
	exposureDetails := _p("Device index is consistent with the above section 'Participating Devices'")
	var exposureListItems []string
	for _, v := range exposureQueryResults {
		var deviceMake string
		if v[3] == "1" {
			deviceMake = "Android"
		} else {
			deviceMake = "iOS"
		}
		exposureListItems = append(exposureListItems, fmt.Sprintf("Exposed to Device %s, %s, average RSSI %s, Contact Duration %s milliseconds, deviceID %s, RPI: %s", v[1], deviceMake, v[2], v[0], v[4], v[5]))
	}
	exposureDetails += _p(fmt.Sprintf("There are %d contacts on record", len(exposureListItems)))
	exposureDetails += _ul(exposureListItems...)

	content = append(content, "Exposure Details", exposureDetails)
	// get other devices
	content = append(content, "Other devices", _h4("NOPE")+_p("yes"))
	return _htmlify(content)
}

func _reportSessionWithoutSessionID(db *sql.DB) string {
	var (
		err                 error
		content             []string
		q                   string
		additionalItems     string
		additionalQResultls []([]string)
	)
	db, err = sql.Open("mysql", url)
	defer fmt.Println("db closed")
	defer db.Close()
	if err != nil {
		fmt.Println(err.Error())
		return err.Error()
	}
	// about session counts
	q = `select count(*) from Test_Sessions;`
	aboutSessions_1 := _dangerouslyQueryForOneNumber(db, q)
	q = `select count(*) from Test_Sessions where isActive = 1;`
	aboutSessions_2 := _dangerouslyQueryForOneNumber(db, q)
	q = `select count(*) from Test_Sessions where isActive = 0;`
	aboutSessions_3 := _dangerouslyQueryForOneNumber(db, q)
	aboutSessions := fmt.Sprintf(`
	There are in total %s sessions in database, of which %s are active and %s are inactive
	`, aboutSessions_1, aboutSessions_2, aboutSessions_3)
	content = append(content, "No session ID provided!! Here is general session information:", aboutSessions)
	// addiontal session details
	q = `
select aa.sessionID, aa.isActive, b.inactiveDeviceCount, b.activeDeviceCount from (select * from Test_Sessions where isActive = 1 order by sessionID desc limit 10) aa inner join (
select sessionID , (select count(*) from Test_hasDevice where sessionID = a.sessionID  and endTime is not null ) as inactiveDeviceCount, (select count(*) from Test_hasDevice where sessionID = a.sessionID and endTime is null) as activeDeviceCount from Test_hasDevice a)b on aa.sessionID = b.sessionID
union
select aa.sessionID, aa.isActive, b.inactiveDeviceCount, b.activeDeviceCount from (select * from Test_Sessions where isActive = 0 order by sessionID desc limit 10) aa inner join (
select sessionID , (select count(*) from Test_hasDevice where sessionID = a.sessionID  and endTime is not null ) as inactiveDeviceCount, (select count(*) from Test_hasDevice where sessionID = a.sessionID and endTime is null) as activeDeviceCount from Test_hasDevice a)b on aa.sessionID = b.sessionID;
`
	additionalQResultls = _dangerouslyQueryForNNumberForMultipleLines(4, db, q)
	fmt.Println(additionalQResultls)
	additionalItemsActive := make([]string, 0)
	additionalItemsInactive := make([]string, 0)
	for _, v := range additionalQResultls {
		if v[1] == "0" {
			// inactive
			additionalItemsInactive = append(additionalItemsInactive,
				fmt.Sprintf(`session ID %s, %s inactive devices, %s active devices`, v[0], v[2], v[3]))
		} else if v[1] == "1" {
			// active
			additionalItemsActive = append(additionalItemsActive,
				fmt.Sprintf(`session ID %s, %s active devices, %s inactive devices`, v[0], v[2], v[3]))
		}
	}
	additionalItems = _h4("Most Recent Acitve Sessions:") +
		_ul(additionalItemsActive...) +
		_h4("Most Recent Inactive Sessions: ") +
		_ul(additionalItemsInactive...)
	content = append(content, "Additional Items", additionalItems)
	return _htmlify(content)
}

// resp += ` No sessionID. Please consider appending "?sessionID=1&deviceIndex=1" at the end of the url for specific session or specific device. Maybe I should list out all sessionIDs, and show how many sessions are ongoing and how many are inactive. Maybe also show stats about when those session begin and end. As well as number of ongoing devices and stuff. `

func _reportSessionWithSessionID(sessionID string, db *sql.DB) string {
	var (
		q1            string
		content       []string
		sessionStatus string
		deviceIndex   string = "0"
		deviceCount   string
		deviceStatus  []string // total
	)
	q1 = ` select isActive from Test_Sessions where sessionID=?; `
	isActive := _dangerouslyQueryForOneNumber(db, q1, sessionID)

	if isActive == "1" {
		sessionStatus = "Active, expect incomplete data"
	} else {
		sessionStatus = "Inactive, data can be complete"
	}
	content = append(content, fmt.Sprintf("Session %s Status", sessionID), sessionStatus)

	// get device status
	deviceStatus = nil
	// get total devices
	q1 = ` select count(*) from Test_hasDevice where sessionID = ?; `
	deviceCount = _dangerouslyQueryForOneNumber(db, q1, sessionID)
	fmt.Println("device count" + deviceCount)
	deviceStatus = append(deviceStatus, deviceCount)
	// get inactive device counts
	q1 = ` select count(*) from Test_hasDevice where sessionID = ? and endTime is not null; `
	deviceCount = _dangerouslyQueryForOneNumber(db, q1, sessionID)
	fmt.Println("device count" + deviceCount)
	deviceStatus = append(deviceStatus, deviceCount)
	// get acitve devices counts
	q1 = ` select count(*) from Test_hasDevice where sessionID = ? and endTime is null; `
	deviceCount = _dangerouslyQueryForOneNumber(db, q1, sessionID)
	fmt.Println("device count" + deviceCount)
	deviceStatus = append(deviceStatus, deviceCount)
	deviceStatusInterface := make([]interface{}, 3)
	for i := 0; i < 3; i++ {
		deviceStatusInterface[i] = deviceStatus[i]
	}
	deviceStatusContent := fmt.Sprintf("There are, in total, %s devices , %s are inactive and %s are active", deviceStatusInterface...)
	deviceStatusContent += _h4("Participating Devices")
	// query for inactive devices first
	q := `
	select deviceIndex, isAndroid, deviceID from Test_hasDevice where sessionID = ? and endTime is not null;
	`
	var participatingDevices []string
	var (
		selfDeviceID, selfIsAndroid string
	)
	participatingDeviceContent_out := _dangerouslyQueryForNNumberForMultipleLines(
		3, db, q, sessionID)
	for _, v := range participatingDeviceContent_out {
		var (
			vvvv, vv, vvv string
		)
		vvvv = "Inactive"
		if v[1] == "1" {
			vv = "Android"
		} else {
			vv = "iOS"
		}
		if string(v[0]) == deviceIndex {
			vvv = _bold("SELF")
			selfDeviceID = v[2]
			selfIsAndroid = v[1]
		}
		participatingDevices = append(participatingDevices,
			fmt.Sprintf("%s device index %s, %s, %s %s", vvvv,
				_a(fmt.Sprintf("SessionReport?deviceIndex=%s&sessionID=%s", v[0], sessionID), v[0]),
				vv, v[2], vvv))
	}
	deviceStatusContent += _ul(participatingDevices...)

	// query for active devices
	participatingDevices = nil
	q = `
	select deviceIndex, isAndroid, deviceID from Test_hasDevice where sessionID = ? and endTime is null;
	`
	participatingDeviceContent_out = _dangerouslyQueryForNNumberForMultipleLines(
		3, db, q, sessionID)

	for _, v := range participatingDeviceContent_out {
		var (
			vvvv, vv, vvv string
		)
		vvvv = "Active"
		if v[1] == "1" {
			vv = "Android"
		} else {
			vv = "iOS"
		}
		if string(v[0]) == deviceIndex {
			vvv = _bold("SELF")
			selfDeviceID = v[2]
			selfIsAndroid = v[1]
		}
		participatingDevices = append(participatingDevices,
			fmt.Sprintf("%s device index %s, %s, %s %s", vvvv,
				_a(fmt.Sprintf("SessionReport?deviceIndex=%s&sessionID=%s", v[0], sessionID), v[0]),
				vv, v[2], vvv))
	}
	deviceStatusContent += _ul(participatingDevices...)
	content = append(content, fmt.Sprintf("Device %s Status", deviceIndex), deviceStatusContent)
	// get exposures
	q = `
	select c.duration, d.deviceIndex, e.RSSI, d.isAndroid, d.deviceID, 
	e.RPI
 from (select a.duration, b.isAndroid, b.deviceID, b.sessionID, b.RPI,
 	a.deviceID as thisDeviceID, a.isAndroid as thisIsAndroid
  from (select * from Test_Exposures where isAndroid = ? and deviceID = ? and sessionID = ?) a 
  inner join Test_RPI b on a.sessionID = b.sessionID and a.RPI = b.RPI )  c inner join 
 Test_hasDevice d on c.sessionID = d.sessionID and 
 c.isAndroid = d.isAndroid and c.deviceID = d.deviceID inner join (select 
 	RPI, isAndroid, deviceID, sessionID, sum(rssi)/count(*) as 
 	RSSI from Test_Exposures_Rssi where RSSI <> 127 group by isAndroid, 
 	deviceID, RPI, sessionID) e on c.sessionid = e.sessionid 
 and c.RPI = e.RPI and c.thisIsAndroid = e.isAndroid and c.thisDeviceID = e.deviceID;
	 `
	exposureQueryResults := _dangerouslyQueryForNNumberForMultipleLines(6, db, q, selfIsAndroid, selfDeviceID, sessionID)
	// I get duration, deviceIndex (other), RSSI (avg), isAndroid, deviceID
	exposureDetails := _p("Device index is consistent with the above section 'Participating Devices'")
	var exposureListItems []string
	for _, v := range exposureQueryResults {
		var deviceMake string
		if v[3] == "1" {
			deviceMake = "Android"
		} else {
			deviceMake = "iOS"
		}
		exposureListItems = append(exposureListItems, fmt.Sprintf("Exposed to Device %s, %s, average RSSI %s, Contact Duration %s milliseconds, deviceID %s, RPI: %s", v[1], deviceMake, v[2], v[0], v[4], v[5]))
	}
	exposureDetails += _p(fmt.Sprintf("There are %d contacts on record", len(exposureListItems)))
	exposureDetails += _ul(exposureListItems...)

	content = append(content, "Exposure Details", exposureDetails)
	// get other devices
	content = append(content, "Other devices", _h4("NOPE")+_p("yes"))
	return _htmlify(content)

}

// input: array of string that follows
// []string{topic1, content1, topic2, content2,...}
// Function is intended to work with even numbered input string
// where the former element in a pair is topic and latter is content for that topic
// output: html string for result rendering
// input array of string in a paired way 0,1 2,3 4,5
func _htmlify(content []string) string {
	pre := `
<!DOCTYPE html>
<html>
<head>
<title>Test Session Report</title>
</head>
<body>
	`
	post := `
</body>
</html>
	`
	var body string
	body = ""
	if len(content)%2 != 0 {
		body = "ERROR! Content is not even numbered"
	} else {
		for i := 0; i < len(content); i += 2 {
			body += fmt.Sprintf(`
			<h3>%s</h3>
			%s
			<hr>
			`, content[i], content[i+1])
		}
	}
	return pre + body + post
}

// html helper
func _bold(s string) string {
	return fmt.Sprintf("<b>%s</b>", s)
}
func _italic(s string) string {
	return fmt.Sprintf("<i>%s</i>", s)
}
func _p(s string) string {
	return fmt.Sprintf("<p>%s</p>", s)
}
func _h4(s string) string {
	return fmt.Sprintf("<h4>%s</h4>", s)
}
func _ul(args ...string) string {
	open := "<ul>"
	close := "</ul>"
	var item string = ""
	for _, v := range args {
		item += `<li>` + string(v) + `</li>`
	}
	return open + item + close
}
func _a(url, text string) string {
	open := fmt.Sprintf("<a href=%s>", url)
	close := "</a>"
	return open + text + close
}

// dangerously query for something I already know exist
/**this function only queries for a number given a string*/
func _dangerouslyQueryForOneNumber(db *sql.DB, s string, args ...interface{}) string {
	var (
		out string
	)
	rows, err := db.Query(s, args...)
	if err != nil {
		fmt.Println(err.Error())
		return err.Error()
	}
	defer rows.Close()
	rows.Next()
	if err := rows.Scan(&out); err != nil {
		fmt.Println(err.Error())
		return err.Error()
	}
	return out
}

/** query for a table of stuff return a string
input: n means number of items to be expected on each line */
func _dangerouslyQueryForNNumberForMultipleLines(n int, db *sql.DB, s string, args ...interface{}) []([]string) {
	var (
		out []([]string)
	)
	rows, err := db.Query(s, args...)
	if err != nil {
		fmt.Println(err.Error())
		out = append(out, []string{err.Error()})
		return out
	}
	defer rows.Close()
	for rows.Next() {
		result := make([]string, n)    // n strings per line
		dest := make([]interface{}, n) // save pointers of string
		for i, _ := range result {
			dest[i] = &result[i] // pointers in dest to scan into
		}
		if err := rows.Scan(dest...); err != nil {
			fmt.Println(err.Error())
			out = append(out, []string{err.Error()})
			return out
		}
		out = append(out, result)
	}
	return out
}

/**function for creating a directed graph using graphviz
 * the query will join */
func _generateEncounterGraph(db *sql.DB, sessionID, deviceIndex string) string {
	var (
		q   string
		out string
	)

	q = `
	select c.scannerIndex, d.advertiserIndex from
	(select b.deviceIndex as "scannerIndex", a.sessionID, a.RPI from
		(select * from Test_Exposures where sessionID = ?) a
	inner join Test_hasDevice b on
	a.isAndroid = b.isAndroid and a.deviceID = b.deviceID and a.sessionID = b.sessionID) c
inner join
	(select d1.sessionID,d1.RPI as "RPI", d2.deviceIndex as "advertiserIndex" from Test_RPI d1 inner join Test_hasDevice d2
	on d1.sessionID = d2.sessionID and d1.isAndroid = d2.isAndroid and d1.deviceID = d2.deviceID) d
on c.sessionID = d.sessionID and c.RPI = d.RPI;
	`
	qResults := _dangerouslyQueryForNNumberForMultipleLines(2, db, q, sessionID)
	g := graphviz.New()
	graph, err := g.Graph()
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := graph.Close(); err != nil {
			log.Fatal(err)
		}
		g.Close()
	}()

	for _, v := range qResults {
		scannerIndex := v[0]
		advertiserIndex := v[1]
		m, err := graph.CreateNode(scannerIndex)
		if scannerIndex == deviceIndex {
			m.SetFontColor("red")
		}
		n, err := graph.CreateNode(advertiserIndex)
		if advertiserIndex == deviceIndex {
			n.SetFontColor("red")
		}
		e, err := graph.CreateEdge("", m, n)
		if err != nil {
			log.Fatal(err)
		}
		e.SetLabel("sees")

	}
	path := fmt.Sprintf("img/SessionID%sDeviceIndex%s.png", sessionID, deviceIndex)
	fmt.Println("path is " + path)
	if err := g.RenderFilename(graph, graphviz.PNG, path); err != nil {
		log.Fatal(err)
	}
	out = fmt.Sprintf("<img src=./%s>", path)
	return out
}
