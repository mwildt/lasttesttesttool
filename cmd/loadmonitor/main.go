package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/mwildt/load-monitor/pkg/connection"
	"github.com/mwildt/load-monitor/pkg/session"
	"github.com/mwildt/load-monitor/pkg/store"
	"github.com/mwildt/load-monitor/pkg/utils"
	"golang.org/x/crypto/bcrypt"
	"log"
	"net"
	"net/http"
	"os"
	"time"
)

var (
	CommitID       = "---"
	BuildBranch    = "development"
	BuildTimestamp = time.Now().UTC().Format(time.RFC3339)
)

type HttpRequestAuthenticator[T any] func(request *http.Request) (T, error)

func AuthenticateAny[T any]() HttpRequestAuthenticator[T] {
	return func(request *http.Request) (res T, err error) {
		return res, err
	}
}

func AuthenticateByKey(hash []byte) HttpRequestAuthenticator[string] {
	type Payload struct {
		Key string `json:"key"`
	}
	return func(request *http.Request) (string, error) {
		var payload Payload
		err := json.NewDecoder(request.Body).Decode(&payload)
		if err != nil {
			return "", err
		}
		if err = bcrypt.CompareHashAndPassword(hash, []byte(payload.Key)); err != nil {
			return "", fmt.Errorf("invalid key")
		} else {
			return "authenticated", nil
		}
	}
}

func randomString(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}

func readOrCreateHashKey(filename string) (hash []byte, err error) {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		f, err := os.Create(filename)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		secret := randomString(16)
		secretBytes := []byte(secret)
		if hash, err = bcrypt.GenerateFromPassword(secretBytes, bcrypt.MinCost); err != nil {
			return nil, err
		} else if _, err = f.Write(hash); err != nil {
			return nil, err
		} else {
			log.Printf("Generate new Access Secret %s\n", secret)
		}
	} else {
		hash, _ = os.ReadFile(filename)
	}
	return hash, nil
}

func FileBasedKeyAuthenticator(filename string) (HttpRequestAuthenticator[string], error) {
	hash, err := readOrCreateHashKey(filename)
	if err != nil {
		return nil, err
	}
	return AuthenticateByKey(hash), err
}

func LoginHandler[T any](store *session.Store[T], authenticator HttpRequestAuthenticator[T], sessionKey string, action Action) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if authentication, err := authenticator(request); err != nil {
			log.Printf("authentication error: %v", err)
			writer.WriteHeader(http.StatusUnauthorized)
			return
		} else if session, err := store.Create(authentication); err != nil {
			log.Printf("Error creating session: %v\n", err)
			writer.WriteHeader(http.StatusInternalServerError)
			return
		} else {
			http.SetCookie(writer, utils.CreateSessionCookie(sessionKey, session))
			action.Run()
			writer.WriteHeader(http.StatusOK)
		}
	}
}

func LogoutHandler[T any](store *session.Store[T], sessionKey string, action Action) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if sid, err := utils.ReadSessionId(request, sessionKey); err != nil {
			writer.WriteHeader(http.StatusBadRequest)
		} else {
			store.Delete(sid)
			http.SetCookie(writer, utils.DeleteCookie(sessionKey))
			action.Run()
			writer.WriteHeader(http.StatusOK)
		}
	}
}

func SystemInfoHandler() http.HandlerFunc {
	info := struct {
		CommitId       string `json:"commitId"`
		BuildTimestamp string `json:"buildTimestamp"`
		BuildBranch    string `json:"buildBranch"`
	}{
		CommitId:       CommitID,
		BuildBranch:    BuildBranch,
		BuildTimestamp: BuildTimestamp,
	}
	return func(writer http.ResponseWriter, request *http.Request) {
		utils.SendJson(writer, request, http.StatusOK, info)
	}
}

func DefaultHandler() http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		//log.Printf("DefaultHandler %s::%s\n", request.Method, request.URL.String())
		writer.WriteHeader(http.StatusOK)
	}
}

func sendJsonEvent[T any](w http.ResponseWriter, event string, data T) error {
	if payload, err := json.Marshal(data); err != nil {
		return err
	} else {
		fmt.Fprintf(w, "event: %s\n", event)
		fmt.Fprintf(w, "data: %s\n\n", base64.StdEncoding.EncodeToString(payload))
	}
	return nil
}

func Noop() Action {
	return func() {}
}

type Action func()

func (action Action) Run() {
	action()
}

func SimpleActionHandler(action Action) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		action.Run()
	}
}

func PostKeyAuthenticationHandler(sessions *session.Store[string], sessionKey string, apikey string) http.HandlerFunc {

	type Paload struct {
		Key string `json:"key"`
	}

	return func(writer http.ResponseWriter, request *http.Request) {
		var payload Paload
		err := json.NewDecoder(request.Body).Decode(&payload)
		if err != nil {
			http.Error(writer, "Invalid JSON", http.StatusBadRequest)
			return
		}

		if payload.Key == apikey {

			session, err := sessions.Create(payload.Key)
			if err != nil {
				log.Printf("Error creating session: %v\n", err)
				http.Error(writer, "Error creating session", http.StatusInternalServerError)
				return
			}

			http.SetCookie(writer, utils.CreateSessionCookie(sessionKey, session))

			go func() {
				time.Sleep(12 * time.Hour)
				sessions.Delete(session.Id)
				fmt.Println("Session gelöscht:", session.Id)
			}()

		} else {
			http.Error(writer, "Not Authorized", http.StatusUnauthorized)
			return
		}
	}
}

type Message[T any] struct {
	Key   string    `json:"key"`
	Value T         `json:"value"`
	Time  time.Time `json:"time"`
}

func createMessage[T any](key string, value T) any {
	return Message[T]{
		Key:   key,
		Value: value,
		Time:  time.Now(),
	}
}

func streamHandler(valueStore *store.Store) http.HandlerFunc {

	return func(writer http.ResponseWriter, request *http.Request) {
		flusher, ok := writer.(http.Flusher)
		if !ok {
			http.Error(writer, "Streaming unsupported", http.StatusInternalServerError)
			return
		}

		timeout := time.NewTimer(10 * time.Second)
		defer timeout.Stop()

		resetTimer := func() {
			timeout.Reset(10 * time.Second)
		}

		bytesBrokerRegistration, storeChannel, err := valueStore.RegisterThrottled(store.All(), 250*time.Millisecond)
		defer valueStore.Cancel(bytesBrokerRegistration)
		if err != nil {
			fmt.Fprintf(writer, "Error registering bytes: %v\n", err)
			writer.WriteHeader(http.StatusInternalServerError)
			return
		}

		writer.Header().Set("Content-Type", "text/event-stream")
		writer.Header().Set("Cache-Control", "no-cache")
		writer.Header().Set("Connection", "keep-alive")

		for key, value := range valueStore.Entries() {
			sendJsonEvent(writer, "store.event", createMessage(key, value))
		}

		flusher.Flush()

		for {
			select {
			case events := <-storeChannel:
				for _, event := range events {
					sendJsonEvent(writer, "store.event", createMessage(event.Key, event.Value))
				}
				flusher.Flush()
				resetTimer()
			case <-timeout.C:
				sendJsonEvent(writer, "ping", createMessage("ping", time.Now()))
				flusher.Flush()
				resetTimer()
			}
		}
	}
}

func requireSession[T any](sessions *session.Store[T], sessionKey string, next http.HandlerFunc) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		cookie, err := request.Cookie(sessionKey)
		if err != nil || cookie.Value == "" {
			http.Error(writer, "Unauthorized: Session fehlt", http.StatusUnauthorized)
			return
		} else if _, exists := sessions.Get(cookie.Value); !exists {
			http.Error(writer, "Unauthorized: Session fehlt", http.StatusUnauthorized)
			return
		}
		next(writer, request)
	}
}

func createListener(valueStore *store.Store, network string, address string) *connection.CountingListener {
	ln, err := net.Listen(network, address)
	if err != nil {
		panic(err)
	}

	return &connection.CountingListener{
		Listener: ln,
		ReadConsumer: func(n int) {
			valueStore.Reduce("bytes.read.count", func(v any) any {
				return int64(n) + v.(int64)
			})
		},
		WriteConsumer: func(n int) {
			valueStore.Reduce("bytes.write.count", func(v any) any {
				return int64(n) + v.(int64)
			})
		},
	}
}

type SensorSessionValue struct{}

func main() {

	valueStore := store.NewStore(map[string]any{
		"session.count":     0,
		"bytes.read.count":  int64(0),
		"bytes.write.count": int64(0),
		"request.count":     0,
	})

	sensorSessionStore := session.NewSessionStore[SensorSessionValue]()
	tcpListener := createListener(valueStore, "tcp", ":8081")

	go runSensorEndpoint(sensorSessionStore, tcpListener, valueStore)
	go runControlEndpoint(valueStore, func() {
		valueStore.Reset()
		sensorSessionStore.Reset()
	}, ":8082")

	select {}
}

func runSensorEndpoint[T any](sessionStore *session.Store[T], listener *connection.CountingListener, valueStore *store.Store) {
	login := LoginHandler(sessionStore, AuthenticateAny[T](), "sid", func() {
		valueStore.Set("session.count", sessionStore.SessionCount())
	})
	logout := LogoutHandler(sessionStore, "sid", func() {
		valueStore.Set("session.count", sessionStore.SessionCount())
	})
	defaultHandler := DefaultHandler()
	srv := &http.Server{
		Addr: listener.Addr().String(),
		Handler: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			valueStore.Reduce("request.count", func(v any) any {
				return v.(int) + 1
			})

			if utils.Match("POST::/login", request) {
				login(writer, request)
			} else if utils.Match("/logout", request) {
				logout(writer, request)
			} else {
				defaultHandler(writer, request)
			}
		}),
	}
	log.Printf("start http sensor-endpoint on %s", listener.Addr().String())
	srv.Serve(listener)
}

func runControlEndpoint(store *store.Store, resetAction Action, addr string) {

	sessionKey := "sessid"
	sessionStore := session.NewSessionStore[string]()
	stream := requireSession(sessionStore, sessionKey, streamHandler(store))
	reset := requireSession(sessionStore, sessionKey, SimpleActionHandler(resetAction))
	logout := LogoutHandler(sessionStore, sessionKey, Noop())
	systemInfo := SystemInfoHandler()

	authenticator, err := FileBasedKeyAuthenticator("./data/auth.sec")
	if err != nil {
		panic(err)
	}

	login := LoginHandler[string](sessionStore, authenticator, sessionKey, Noop())
	static := http.FileServer(http.Dir("./static"))

	log.Printf("start http control-endpoint on %s", addr)

	go http.ListenAndServe(addr, http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if utils.Match("GET::/system-info", request) {
			systemInfo(writer, request)
		} else if utils.Match("POST::/auth", request) {
			login(writer, request)
		} else if utils.Match("/logout", request) {
			logout(writer, request)
		} else if utils.Match("GET::/stream", request) {
			stream(writer, request)
		} else if utils.Match("PATCH::/reset", request) {
			reset(writer, request)
		} else {
			static.ServeHTTP(writer, request)
		}
	}))
}
