package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"

	"github.com/matt/dungeon-revealer-go/internal/auth"
	"github.com/matt/dungeon-revealer-go/internal/config"
	"github.com/matt/dungeon-revealer-go/internal/handler"
	"github.com/matt/dungeon-revealer-go/internal/realtime"
	"github.com/matt/dungeon-revealer-go/internal/store"
)

//go:embed static
var staticFS embed.FS

func main() {
	cfg := config.Parse()

	db, err := store.New(cfg.DataDir)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	a := auth.New(cfg.SessionSecret, cfg.DMPassword, cfg.PlayerPassword)
	maps := store.NewMapStore(cfg.DataDir)
	chat := store.NewChatStore(200)
	broker := realtime.NewBroker()

	authHandler := handler.NewAuthHandler(a)
	homeHandler := handler.NewHomeHandler(a)
	dmHandler := handler.NewDMHandler(maps, a)
	mapHandler := handler.NewMapHandler(maps)
	fogHandler := handler.NewFogHandler(maps, broker)
	tokenHandler := handler.NewTokenHandler(maps, broker)
	chatHandler := handler.NewChatHandler(chat, broker, a)
	notesHandler := handler.NewNotesHandler(db, broker)
	media := store.NewMediaStore(db, cfg.DataDir)
	mediaHandler := handler.NewMediaHandler(media)

	mux := http.NewServeMux()

	// Static files
	staticSub, _ := fs.Sub(staticFS, "static")
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticSub))))

	// SSE events
	mux.Handle("GET /events", a.RequireRole(auth.RolePlayer)(http.HandlerFunc(broker.ServeHTTP)))

	// Auth routes
	mux.HandleFunc("GET /login", authHandler.LoginPage)
	mux.HandleFunc("POST /login", authHandler.LoginSubmit)
	mux.HandleFunc("GET /logout", authHandler.Logout)

	// Player view
	mux.Handle("GET /{$}", a.RequireRole(auth.RolePlayer)(http.HandlerFunc(homeHandler.PlayerPage)))
	mux.Handle("GET /maps/active", a.RequireRole(auth.RolePlayer)(http.HandlerFunc(mapHandler.PlayerMapView)))
	mux.Handle("GET /maps/{id}/image", a.RequireRole(auth.RolePlayer)(http.HandlerFunc(mapHandler.ServeImage)))
	mux.Handle("GET /maps/{id}/fog", a.RequireRole(auth.RolePlayer)(http.HandlerFunc(fogHandler.ServeLive)))
	mux.Handle("GET /chat/messages", a.RequireRole(auth.RolePlayer)(http.HandlerFunc(chatHandler.Messages)))
	mux.Handle("POST /chat/send", a.RequireRole(auth.RolePlayer)(http.HandlerFunc(chatHandler.Send)))
	mux.Handle("POST /chat/name", a.RequireRole(auth.RolePlayer)(http.HandlerFunc(chatHandler.SetName)))
	mux.Handle("GET /chat/name/edit", a.RequireRole(auth.RolePlayer)(http.HandlerFunc(chatHandler.EditNameForm)))
	mux.Handle("GET /notes", a.RequireRole(auth.RolePlayer)(http.HandlerFunc(notesHandler.PlayerList)))
	mux.Handle("GET /notes/search", a.RequireRole(auth.RolePlayer)(http.HandlerFunc(notesHandler.PlayerSearch)))
	mux.Handle("GET /notes/{id}", a.RequireRole(auth.RolePlayer)(http.HandlerFunc(notesHandler.PlayerView)))
	mux.Handle("GET /notes/{id}/edit", a.RequireRole(auth.RolePlayer)(http.HandlerFunc(notesHandler.PlayerEditForm)))
	mux.Handle("POST /notes/{id}", a.RequireRole(auth.RolePlayer)(http.HandlerFunc(notesHandler.PlayerSave)))
	mux.Handle("GET /media/{id}", a.RequireRole(auth.RolePlayer)(http.HandlerFunc(mediaHandler.Serve)))
	mux.Handle("GET /maps/{id}/tokens", a.RequireRole(auth.RolePlayer)(http.HandlerFunc(tokenHandler.ListTokens)))
	mux.Handle("PUT /maps/{id}/tokens/{tokenId}", a.RequireRole(auth.RolePlayer)(http.HandlerFunc(tokenHandler.UpdateToken)))

	// DM view
	mux.Handle("GET /dm", a.RequireRole(auth.RoleAdmin)(http.HandlerFunc(dmHandler.Dashboard)))
	mux.Handle("POST /dm/maps/upload", a.RequireRole(auth.RoleAdmin)(http.HandlerFunc(mapHandler.Upload)))
	mux.Handle("GET /dm/maps", a.RequireRole(auth.RoleAdmin)(http.HandlerFunc(mapHandler.List)))
	mux.Handle("GET /dm/maps/active", a.RequireRole(auth.RoleAdmin)(http.HandlerFunc(mapHandler.ActiveMapView)))
	mux.Handle("POST /dm/maps/{id}/activate", a.RequireRole(auth.RoleAdmin)(http.HandlerFunc(mapHandler.SetActive)))
	mux.Handle("DELETE /dm/maps/{id}", a.RequireRole(auth.RoleAdmin)(http.HandlerFunc(mapHandler.Delete)))
	mux.Handle("GET /dm/maps/{id}/image", a.RequireRole(auth.RoleAdmin)(http.HandlerFunc(mapHandler.ServeImage)))
	mux.Handle("PUT /dm/maps/{id}/fog/progress", a.RequireRole(auth.RoleAdmin)(http.HandlerFunc(fogHandler.SaveProgress)))
	mux.Handle("GET /dm/maps/{id}/fog/progress", a.RequireRole(auth.RoleAdmin)(http.HandlerFunc(fogHandler.ServeProgress)))
	mux.Handle("POST /dm/maps/{id}/fog/push", a.RequireRole(auth.RoleAdmin)(http.HandlerFunc(fogHandler.Push)))
	mux.Handle("GET /dm/maps/{id}/tokens", a.RequireRole(auth.RoleAdmin)(http.HandlerFunc(tokenHandler.ListTokens)))
	mux.Handle("POST /dm/maps/{id}/tokens", a.RequireRole(auth.RoleAdmin)(http.HandlerFunc(tokenHandler.CreateToken)))
	mux.Handle("PUT /dm/maps/{id}/tokens/{tokenId}", a.RequireRole(auth.RoleAdmin)(http.HandlerFunc(tokenHandler.UpdateToken)))
	mux.Handle("DELETE /dm/maps/{id}/tokens/{tokenId}", a.RequireRole(auth.RoleAdmin)(http.HandlerFunc(tokenHandler.DeleteToken)))
	mux.Handle("GET /dm/notes", a.RequireRole(auth.RoleAdmin)(http.HandlerFunc(notesHandler.List)))
	mux.Handle("GET /dm/notes/search", a.RequireRole(auth.RoleAdmin)(http.HandlerFunc(notesHandler.Search)))
	mux.Handle("GET /dm/notes/{id}", a.RequireRole(auth.RoleAdmin)(http.HandlerFunc(notesHandler.View)))
	mux.Handle("GET /dm/notes/{id}/edit", a.RequireRole(auth.RoleAdmin)(http.HandlerFunc(notesHandler.EditForm)))
	mux.Handle("POST /dm/notes/{id}", a.RequireRole(auth.RoleAdmin)(http.HandlerFunc(notesHandler.Save)))
	mux.Handle("DELETE /dm/notes/{id}", a.RequireRole(auth.RoleAdmin)(http.HandlerFunc(notesHandler.Delete)))
	mux.Handle("POST /dm/notes/{id}/lock", a.RequireRole(auth.RoleAdmin)(http.HandlerFunc(notesHandler.LockToggle)))
	mux.Handle("POST /dm/media/upload", a.RequireRole(auth.RoleAdmin)(http.HandlerFunc(mediaHandler.Upload)))
	mux.Handle("GET /dm/media", a.RequireRole(auth.RoleAdmin)(http.HandlerFunc(mediaHandler.List)))
	mux.Handle("DELETE /dm/media/{id}", a.RequireRole(auth.RoleAdmin)(http.HandlerFunc(mediaHandler.Delete)))

	// Wrap all routes with auth middleware
	server := a.Middleware(mux)

	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("Dungeon Revealer starting on http://localhost%s", addr)
	if cfg.DMPassword != "" {
		log.Printf("DM password is set")
	} else {
		log.Printf("No DM password — all users have admin access")
	}

	if err := http.ListenAndServe(addr, server); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
