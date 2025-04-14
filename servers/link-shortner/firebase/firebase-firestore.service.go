package link_shorner_firebase

import (
	"cloud.google.com/go/firestore"
	"github.com/rpsoftech/golang-servers/utility/firebase"
)

// "github.com/rpsoftech/bullion-server/src/utility/firebase"

type firebaseFirestoreService struct {
	db *firestore.Client
}

var firebaseFirestoreDatabaseService *firebaseFirestoreService

func GetFirebaseFirestoreDatabase() *firebaseFirestoreService {
	if firebaseFirestoreDatabaseService == nil {
		firebaseFirestoreDatabaseService = &firebaseFirestoreService{
			db: firebase.FirebaseFirestore,
		}
		println("Firebase Realtime Database Service Initialized")
	}
	return firebaseFirestoreDatabaseService
}

func (s *firebaseFirestoreService) SetPublicData(urlString string, data interface{}) error {
	return s.setPrivateData("urls", urlString, data)
}

func (s *firebaseFirestoreService) setPrivateData(collection string, doc string, data interface{}) error {
	_, err := s.db.Collection(collection).Doc(doc).Set(firebase.FirebaseCtx, data)
	return err
}
func (s *firebaseFirestoreService) GetShortUrl(doc string) (map[string]interface{}, error) {
	return s.GetPublicData("urls", doc)
	// da, err := s.db.Collection(colletion).Doc(doc).Get(FirebaseCtx)
	// if err != nil {
	// 	return nil, err
	// }
	// return da.Data(), err
}
func (s *firebaseFirestoreService) GetPublicData(colletion string, doc string) (map[string]interface{}, error) {
	da, err := s.db.Collection(colletion).Doc(doc).Get(firebase.FirebaseCtx)
	if err != nil {
		println(err)
		return nil, err
	}
	return da.Data(), err
}
