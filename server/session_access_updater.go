package server

import (
	"boivie/ubergang/server/models"
	"time"
)

func (s *Server) sessionAccessUpdater() {
	toUpdate := make(map[string]time.Time)
	flush := make(chan *models.Session, 1)
	for {
		select {
		case c := <-s.updateAccessed:
			if _, found := toUpdate[c.Id]; !found {
				go func() {
					time.Sleep(1 * time.Minute)
					flush <- c
				}()
			}
			toUpdate[c.Id] = time.Now()
		case session := <-flush:
			// for idx := range u.Sessions {
			// 	if u.Sessions[idx].ID == session.ID {
			// 		u.Sessions[idx].AccessedAt = toUpdate[session.ID].Format(time.RFC3339)
			// 	}
			// }
			// tx := s.storage.StartTransaction("Update access time")
			// tx.UpdateUser(u)
			// tx.Commit()
			delete(toUpdate, session.Id)
		}
	}
}
