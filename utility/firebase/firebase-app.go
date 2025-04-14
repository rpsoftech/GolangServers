package firebase

import (
	"context"
	"log"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"firebase.google.com/go/v4/db"
	"firebase.google.com/go/v4/messaging"
	"github.com/rpsoftech/golang-servers/env"
	"google.golang.org/api/option"
)

var firebaseApp *firebase.App
var FirebaseCtx context.Context
var FirebaseDb *db.Client
var FirebaseFirestore *firestore.Client
var FirebaseAuth *auth.Client
var FirebaseFCM *messaging.Client

func DeferFunction() {
	println("Firebase Defer Function")
	// FirebaseFirestore.Close()
	FirebaseFirestore.Close()
	// FirebaseAuth.Close()
	// FirebaseFCM.Close()
}

func init() {
	if env.Env.APP_ENV == env.APP_ENV_DEVELOPE {
		return
	}

	FirebaseCtx = context.Background()
	firebaseJson := env.Env.GetEnv(env.FIREBASE_JSON_STRING_KEY)
	if firebaseJson == "" {
		panic("Please Pass Valid Firebase JSON string as ENV KEY " + env.FIREBASE_JSON_STRING_KEY)
	}
	opt := option.WithCredentialsJSON([]byte(firebaseJson))

	app, err := firebase.NewApp(FirebaseCtx, nil, opt)
	if err != nil {
		log.Fatalf("error initializing app: %v\n", err)
	}
	firebaseApp = app
	log.Print("Firebase App initialized")
	firebaseDatabaseUrl := env.Env.GetEnv(env.FIREBASE_DATABASE_URL_KEY)
	if firebaseDatabaseUrl == "" {
		panic("Please Pass Valid Firebase Database URL string as ENV KEY " + env.FIREBASE_DATABASE_URL_KEY)
	}
	firebaseDb, err := firebaseApp.DatabaseWithURL(FirebaseCtx, firebaseDatabaseUrl)
	if err != nil {
		log.Fatalf("error initializing Firebase Database: %v\n", err)
	}
	FirebaseDb = firebaseDb
	firestoreDb, err := firebaseApp.Firestore(FirebaseCtx)
	if err != nil {
		log.Fatalf("error initializing Firebase Database: %v\n", err)
	}
	FirebaseFirestore = firestoreDb
	firestoreAuth, err := firebaseApp.Auth(FirebaseCtx)
	if err != nil {
		log.Fatalf("error initializing Firebase Database: %v\n", err)
	}
	FirebaseAuth = firestoreAuth
	fcm, err := firebaseApp.Messaging(FirebaseCtx)
	if err != nil {
		log.Fatalf("error initializing Firebase Database: %v\n", err)
	}
	FirebaseFCM = fcm
	println("Firebase App Initialize")
}

// ctx := context.Background()
// conf := &firebase.Config{
//         DatabaseURL: "https://databaseName.firebaseio.com",
// }
// // Fetch the service account key JSON file contents
// opt := option.WithCredentialsFile("path/to/serviceAccountKey.json")

// // Initialize the app with a service account, granting admin privileges
// app, err := firebase.NewApp(ctx, conf, opt)
// if err != nil {
//         log.Fatalln("Error initializing app:", err)
// }

// client, err := app.Database(ctx)
// if err != nil {
//         log.Fatalln("Error initializing database client:", err)
// }

// // As an admin, the app has access to read and write all data, regradless of Security Rules
// ref := client.NewRef("restricted_access/secret_document")
// var data map[string]interface{}
// if err := ref.Get(ctx, &data); err != nil {
//         log.Fatalln("Error reading from database:", err)
// }
// fmt.Println(data)
