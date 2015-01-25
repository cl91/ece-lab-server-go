package main

import (
	"encoding/json"
	"crypto/rand"
	"encoding/base64"
	"net/http"
)

type AuthReply struct {
	Name string `json:"name"`
	Auth string `json:"auth"`
	Type string `json:"type"`
}

func AuthHandler(req Request) Response {
	namev, ok := req.query["name"]
	if !ok {
		return Response { code : BadRequest, msg : "Username required" }
	}
	passv, ok := req.query["pass"]
	if !ok {
		return Response { code : BadRequest, msg : "Password required" }
	}
	name := namev[0]
	pass := passv[0]
	if reply, _ := req.db.Sismember("users", name); !reply {
		return Response { code : Unauthorised, msg : "Invalid user name: " + name }
	}
	realpass, _ := req.db.Hget("user:"+name, "pass")
	if pass != realpass {
		return Response { code : Unauthorised, msg : "Invalid password for user: " + name }
	}

	// Generate a 44 byte, base64 encoded output
	token, err := GenerateRandomString(32)
	if err != nil {
		return Response { code : ServerError, msg : "Failed to generate authorisation token" }
	}

	usertype, _ := req.db.Hget("user:"+name, "type")

	reply := AuthReply { Name : name, Auth : token, Type: usertype }
	result, err := json.Marshal(reply)
	if err != nil {
		return Response { code : ServerError, msg : "Error marshalling json object: " + err.Error() }
	}

	req.db.Hset("user:"+name, "auth", token)
	req.db.Hset("auth", token, name)
	cookie := http.Cookie { Name : "auth", Value : token }

	return Response { msg: string(result), cookie : &cookie }
}

// GenerateRandomBytes returns securely generated random bytes. 
// It will return an error if the system's secure random
// number generator fails to function correctly, in which
// case the caller should not continue.
func GenerateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	// Note that err == nil only if we read len(b) bytes.
	if err != nil {
		return nil, err
	}

	return b, nil
}

// GenerateRandomString returns a URL-safe, base64 encoded
// securely generated random string.
// It will return an error if the system's secure random
// number generator fails to function correctly, in which
// case the caller should not continue.
func GenerateRandomString(s int) (string, error) {
	b, err := GenerateRandomBytes(s)
	return base64.URLEncoding.EncodeToString(b), err
}

func PasswdHandler(req Request) Response {
	user := req.user
	student := req.student
	if user == "" && student == "" {
		return Response { code : BadRequest, msg : "Invalid user" }
	}
	oldpassv, ok := req.query["oldpass"]
	if !ok {
		return Response { code : BadRequest, msg : "Old password required" }
	}
	newpassv, ok := req.query["newpass"]
	if !ok {
		return Response { code : BadRequest, msg : "New password required" }
	}
	oldp := oldpassv[0]
	newp := newpassv[0]
	oldpass := ""
	if user != "" {
		oldpass, _ = req.db.Hget("user:"+user, "pass")
	} else if student != "" {
		oldpass, _ = req.db.Hget("student:"+student, "pass")
	}
	if oldpass != oldp {
		return Response { code : BadRequest, msg : "Incorrect old password" }
	}
	if user != "" {
		req.db.Hset("user:"+user, "pass", newp)
	} else if student != "" {
		req.db.Hset("student:"+student, "pass", newp)
	}
	return Response { msg : "Password changed." }
}
