package bullion_main_server_services

import (
	"firebase.google.com/go/v4/db"
	"github.com/rpsoftech/golang-servers/utility/firebase"
)

type firebaseDatabaseService struct {
	db *db.Client
}

var firebaseRealTimeDatabaseService *firebaseDatabaseService

func getFirebaseRealTimeDatabase() *firebaseDatabaseService {
	if firebaseRealTimeDatabaseService == nil {
		firebaseRealTimeDatabaseService = &firebaseDatabaseService{
			db: firebase.FirebaseDb,
		}
		println("Firebase Realtime Database Service Initialized")
	}
	return firebaseRealTimeDatabaseService
}

func (s *firebaseDatabaseService) SetPublicData(bullionId string, path []string, data interface{}) error {
	return s.setPrivateData("bullions/"+bullionId, path, data)
}

func (s *firebaseDatabaseService) setPrivateData(base string, path []string, data interface{}) error {
	ref := s.db.NewRef(base)
	for _, child := range path {
		ref = ref.Child(child)
	}
	return ref.Set(firebase.FirebaseCtx, data)
}
func (s *firebaseDatabaseService) GetPublicData(bullionId string, path []string, data interface{}) error {
	return s.GetData("bullions/"+bullionId, path, data)
}
func (s *firebaseDatabaseService) GetData(base string, path []string, v interface{}) error {
	ref := s.db.NewRef(base)
	for _, child := range path {
		ref = ref.Child(child)
	}
	return ref.Get(firebase.FirebaseCtx, v)
}
