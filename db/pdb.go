package db

import (
	"bufio"
	"crypto/rand"
	"crypto/sha1"
	"database/sql"
	"errors"
	"fmt"
	"imgturtle/misc"
	"os"

	"github.com/fatih/color"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/pbkdf2"
)

var db *sql.DB

// Start is the database package launch method
// it enters or fetches the data required for the database
func Start() {
	/*
	 * allow user to enter db data
	 * used instead of environment variables
	 * if there are none
	 * since the service is open source
	 */
	var (
		uname string
		pw    string
		name  string
	)
	if os.Getenv("DB_UNAME") == "" && os.Getenv("DB_NAME") == "" {
		reader := bufio.NewReader(os.Stdin)
		color.Cyan("Enter db user name: ")
		uname, _ = reader.ReadString('\n')

		color.Cyan("Enter db pw: ")
		pw, _ = reader.ReadString('\n')

		color.Cyan("Enter db name: ")
		name, _ = reader.ReadString('\n')
	} else {
		uname = os.Getenv("DB_UNAME")
		pw = os.Getenv("DB_PW")
		name = os.Getenv("DB_NAME")
	}
	var err error
	db, err = sql.Open("postgres",
		"user="+uname+
			" password="+pw+
			" dbname="+name+
			" sslmode=disable")

	if err != nil {
		misc.PrintMessage(1, "db  ", "pdb.go", "Start()", "PostgreSQL config could not be established\n."+err.Error())
		return
	}
	// test connection
	err = db.Ping()
	if err != nil { // connection not successful
		misc.PrintMessage(1, "db  ", "pdb.go", "Start()", "Database connection not working.\n"+err.Error())
		var rundb string
		reader := bufio.NewReader(os.Stdin)
		color.Cyan("Do you want to run the server without a working database module? (y/n) ")
		rundb, _ = reader.ReadString('\n')
		if rundb != "y\n" && rundb != "Y\n" {
			os.Exit(-1)
		}
	}
}

// CheckUserCredentials handles the database part of the
// login process
func CheckUserCredentials(ue string, pwd string) (bool, error) {
	rows, err := db.Query("select user_name, user_email, user_pw, user_hash from imgturtle.user where user_name='" + ue + "' or user_email='" + ue + "'")
	if err != nil {
		misc.PrintMessage(1, "db  ", "pdb.go", "CheckUserCredentials()", err.Error())
		return false, err
	}

	if rows != nil {
		defer rows.Close()

		var (
			fUname string
			fEmail string
			fPw    string
			fHash  string
		)
		if rows.Next() {
			err := rows.Scan(&fUname, &fEmail, &fPw, &fHash)
			if err != nil {
				misc.PrintMessage(1, "db  ", "pdb.go", "CheckUserCredentials()", "Fetched values could not be scanned.\n"+err.Error())
				return false, err
			}
			if fPw == string(pbkdf2.Key([]byte(pwd), []byte(fHash), 4096, 32, sha1.New)) {
				misc.PrintMessage(0, "db  ", "pdb.go", "CheckUserCredentials()", "User "+fUname+" entered a valid password.")
				return true, nil
			}
			misc.PrintMessage(1, "db  ", "pdb.go", "CheckUserCredentials()", "User "+fUname+" entered an invalid password.")
			return false, errors.New("Incorrect username or password.")
		}
		misc.PrintMessage(0, "db  ", "pdb.go", "CheckUserCredentials()", "User "+ue+" could not be found.")
		return false, errors.New("Incorrect username or password.")
	}
	return false, nil
}

// InsertNewUser handles the database part of the process of
// registering a new user
func InsertNewUser(uname string, pwd string, email string) error {
	rows, err := db.Query("select user_name, user_email from imgturtle.user where user_name='" + uname + "' or user_email='" + email + "'")
	if err != nil {
		misc.PrintMessage(1, "db  ", "pdb.go", "InsertNewUser()", err.Error())
	}
	if rows != nil {
		defer rows.Close()
		var (
			funame string
			femail string
		)
		for rows.Next() {
			err := rows.Scan(&funame, &femail)
			if err != nil {
				misc.PrintMessage(1, "db  ", "pdb.go", "InsertNewUser()", "Fetched values could not be scanned.\n"+err.Error())
				color.Red(err.Error())
				return err
			}
			if funame == uname && femail == email {
				return errors.New("User name '" + uname + "' and e-mail address '" + email + "' in use.")
			} else if funame == uname {
				return errors.New("User name '" + uname + "' in use.")
			} else if femail == email {
				return errors.New("E-mail address '" + email + "'in use.")
			}
		}
	}

	b := make([]byte, 32)
	rand.Read(b)
	salt := fmt.Sprintf("%x", b)

	epw := pbkdf2.Key([]byte(pwd), []byte(salt), 4096, 32, sha1.New)

	stmt, err := db.Prepare("INSERT INTO imgturtle.user(user_name,user_pw,user_email,user_hash) VALUES($1,$2,$3,$4)")
	if err != nil {
		return err
	}
	_, err = stmt.Exec(uname, epw, email, salt)
	if err != nil {
		return err
	}
	misc.PrintMessage(0, "db  ", "pdb.go", "InsertNewUser()", "New user "+uname+" with email "+email+" has been created and stored in the database.")
	return nil
}

func CheckIfUserExists(name string) (bool, error) {
	rows, err := db.Query("select 1 from imgturtle.User where user_name='" + name + "'")
	if err != nil {
		misc.PrintMessage(1, "db  ", "pdb.go", "CheckIfUserExists()", "500 "+err.Error())
		return false, err
	}
	defer rows.Close()
	var num int
	if rows == nil {
		return false, nil
	} else {
		for rows.Next() {
			err := rows.Scan(&num)
			if err != nil {
				return false, err
			}
			return true, nil
		}
	}
	return false, nil
}

func CheckIfAlreadyFollowing(initiatorName string, receiverName string) (bool, error) {
	rows, err := db.Query("select 1 from imgturtle.Following where user1_name='" + initiatorName + "' and user2_name='" + receiverName + "'")
	if err != nil {
		misc.PrintMessage(1, "db  ", "pdb.go", "CheckIfAlreadyFollowing()", "500 "+err.Error())
		return false, err
	}
	defer rows.Close()
	var num int
	if rows == nil {
		return false, nil
	} else {
		for rows.Next() {
			err := rows.Scan(&num)
			if err != nil {
				return false, err
			}
			return true, nil
		}
	}
	return false, nil
}

func CreateFollowerRelationShip(initiatiorName string, receiverName string) error {
	stmt, err := db.Prepare("INSERT INTO imgturtle.Following(user1_name, user2_name) VALUES($1, $2)")
	if err != nil {
		return err
	}
	_, err = stmt.Exec(initiatiorName, receiverName)
	if err != nil {
		return err
	}
	return nil
}

func CheckIfImageExists(id string) (bool, string, string, string, int, error) {
	rows, err := db.Query("select image_id, image_title, image_path from imgturtle.img where image_id='" + id + "'")
	if err != nil {
		misc.PrintMessage(1, "db  ", "pdb.go", "CheckIfImageExists()", "500 "+err.Error())
	}
	var (
		fID  string
		ftit string
		fpat string
	)
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			err := rows.Scan(&fID, &ftit, &fpat)
			if err != nil {
				misc.PrintMessage(1, "db  ", "pdb.go", "CheckIfImageExists()", "Fetched values could not be scanned.\n"+err.Error())
				return false, "", "", "", 500, err
			}
		}
		if fID == id {
			return true, fpat, fID, ftit, 200, nil
		}
	}
	return false, "", "", "", 404, errors.New("No image with id " + id + " could be found.")
}

func CheckIfImageIDInUse(id string) error {
	rows, err := db.Query("select image_id from imgturtle.img where image_id='" + id + "'")
	if err != nil {
		misc.PrintMessage(1, "db  ", "pdb.go", "CheckImageID()", err.Error())
	}

	if rows != nil {
		defer rows.Close()

		var fID string
		for rows.Next() {
			err := rows.Scan(&fID)
			if err != nil {
				misc.PrintMessage(1, "db  ", "pdb.go", "CheckImageID()", "Fetched values could not be scanned.\n"+err.Error())
				return err
			}
			if fID == id {
				return errors.New("Image ID '" + id + "' in use.")
			}
		}
	}
	return nil
}

// StoreImage stores all of an image's information in the database
func StoreImage(id string, title string, imgPath string, ext string /*, desc string, uploader_id string, uploader_name string*/) error {
	stmt, err := db.Prepare("INSERT INTO imgturtle.Img(image_id, image_title, image_path, image_f_ext) VALUES($1, $2, $3, $4)")
	if err != nil {
		return err
	}
	_, err = stmt.Exec(id, title, imgPath, ext)
	if err != nil {
		return err
	}
	return nil
}
