package rest

import (
	"boivie/ubergang/server/common"
	"boivie/ubergang/server/models"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// handlePublicBackup streams a snapshot of the whole database. It holds every
// secret that the server has, so it requires the admin scope, and each backup
// is logged.
func (s *ApiModule) handlePublicBackup(w http.ResponseWriter, r *http.Request, key *models.ApiKey) {
	s.log.Infof("API key %s (%s) is downloading a database backup, from %s",
		key.Id, key.Name, common.ReadUserIP(r, s.trustProxyHeaders))

	started := false
	err := s.db.Backup(w, func(size int64) {
		started = true
		filename := fmt.Sprintf("ubergang-%s.db", time.Now().UTC().Format("20060102-150405"))
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
		w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusOK)
	})
	if err != nil {
		s.log.Warnf("Database backup failed: %v", err)
		// Once streaming has started, the short body tells the client that
		// the backup is incomplete, as it doesn't match Content-Length.
		if !started {
			writePublicError(w, http.StatusInternalServerError, publicErrInternal, "The backup failed")
		}
	}
}
