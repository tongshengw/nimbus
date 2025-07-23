package middle

import (
	"context"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/tongshengw/nimbus/backend/sectionleader/internal/shared"
)

func NewJwt(id shared.MachineUUID, secretKey string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256,
		jwt.MapClaims{
			"machineId": id.String(),
			"iat":       time.Now().Unix(),
		})

	tokenStr, err := token.SignedString([]byte(secretKey))
	if err != nil {
		logrus.Errorf("new jwt signedstring failed: %v", err)
		return "", err
	}

	return tokenStr, nil
}

func CheckJwt(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		data, ok := r.Context().Value(CommonContextDataKey).(CommonContextData)
		if !ok {
			logrus.Errorf("common context data not ok: %v", data)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		secretKey := data.SecretKey

		tokenString := r.Header.Get("Authorization")

		parserOpt := jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()})
		parser := jwt.NewParser(parserOpt)

		token, err := parser.Parse(tokenString, func(token *jwt.Token) (any, error) {
			return []byte(secretKey), nil
		})
		if err != nil {
			logrus.Errorf("auth failed: %v", err)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			machineIdStr, ok := claims["machineId"].(string)
			if !ok {
				logrus.Errorf("token parse error")
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
			logrus.Infof("jwt parsed for %s", machineIdStr)

			newUUID, err := uuid.Parse(machineIdStr)
			if err != nil {
				logrus.Errorf("uuid frombytes error: %v", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			machineId := shared.MachineUUID(newUUID)

			newCtx := context.WithValue(r.Context(), MachineIdContextDataKey, machineId)
			next.ServeHTTP(w, r.WithContext(newCtx))
		} else {
			logrus.Errorf("token parse error")
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	})
}
