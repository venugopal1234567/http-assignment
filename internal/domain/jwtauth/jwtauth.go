package jwtauth

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/go-redis/redis"
	"github.com/gofrs/uuid"
)

type JwtAuthService interface {
	CreateToken(userid uint64) (*TokenDetails, error)
	CreateAuth(userid uint64, td *TokenDetails) error
	VerifyToken(tokenString string) (*jwt.Token, error)
	FetchAuth(authD *AccessDetails) (uint64, error)
	ExtractTokenMetadata(tokenString string) (*AccessDetails, error)
	DeleteAuth(givenUuid string) (int64, error)
	ExtractToken(r *http.Request) string
	TokenValid(tokenString string) error
	VerifyRefreshToken(refreshToken string) (*jwt.Token, error)
	RefreshToken(token *jwt.Token) (uint64, *TokenDetails, error)
}

type jwtAuthService struct {
	accessKey  string
	refreshKey string
	client     *redis.Client
}

func NewJwtAuthService(ak string, rk string, rc *redis.Client) JwtAuthService {
	return &jwtAuthService{
		accessKey:  ak,
		refreshKey: rk,
		client:     rc,
	}
}
func (j *jwtAuthService) CreateToken(userid uint64) (*TokenDetails, error) {

	td := &TokenDetails{}
	td.AtExpires = time.Now().Add(time.Minute * 15).Unix()
	accessUuid, _ := uuid.NewV4()
	td.AccessUuid = accessUuid.String()
	td.RtExpires = time.Now().Add(time.Hour * 24 * 7).Unix()
	refreshUuid, _ := uuid.NewV4()
	td.AccessUuid = refreshUuid.String()

	var err error
	atClaims := jwt.MapClaims{}
	atClaims["authorized"] = true
	atClaims["access_uuid"] = td.AccessUuid
	atClaims["user_id"] = userid
	atClaims["exp"] = td.AtExpires
	at := jwt.NewWithClaims(jwt.SigningMethodHS256, atClaims)
	td.AccessToken, err = at.SignedString([]byte(j.accessKey))
	if err != nil {
		return nil, err
	}
	//Creating Refresh Token

	rtClaims := jwt.MapClaims{}
	rtClaims["refresh_uuid"] = td.RefreshUuid
	rtClaims["user_id"] = userid
	rtClaims["exp"] = td.RtExpires
	rt := jwt.NewWithClaims(jwt.SigningMethodHS256, rtClaims)
	td.RefreshToken, err = rt.SignedString([]byte(j.refreshKey))
	if err != nil {
		return nil, err
	}
	return td, nil
}

func (j *jwtAuthService) CreateAuth(userid uint64, td *TokenDetails) error {
	at := time.Unix(td.AtExpires, 0) //converting Unix to UTC(to Time object)
	rt := time.Unix(td.RtExpires, 0)
	now := time.Now()

	errAccess := j.client.Set(td.AccessUuid, strconv.Itoa(int(userid)), at.Sub(now)).Err()
	if errAccess != nil {
		return errAccess
	}
	errRefresh := j.client.Set(td.RefreshUuid, strconv.Itoa(int(userid)), rt.Sub(now)).Err()
	if errRefresh != nil {
		return errRefresh
	}
	return nil
}

func (j *jwtAuthService) FetchAuth(authD *AccessDetails) (uint64, error) {
	userid, err := j.client.Get(authD.AccessUuid).Result()
	if err != nil {
		return 0, err
	}
	userID, _ := strconv.ParseUint(userid, 10, 64)
	return userID, nil
}

func (j *jwtAuthService) VerifyToken(tokenString string) (*jwt.Token, error) {

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		//Make sure that the token method conform to "SigningMethodHMAC"
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(j.accessKey), nil
	})
	if err != nil {
		return nil, err
	}
	return token, nil
}

func (j *jwtAuthService) ExtractTokenMetadata(tokenString string) (*AccessDetails, error) {
	token, err := j.VerifyToken(tokenString)
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if ok && token.Valid {
		accessUuid, ok := claims["access_uuid"].(string)
		if !ok {
			return nil, fmt.Errorf("faled to get user id from claims")
		}
		userId, err := strconv.ParseUint(fmt.Sprintf("%.f", claims["user_id"]), 10, 64)
		if err != nil {
			return nil, err
		}
		return &AccessDetails{
			AccessUuid: accessUuid,
			UserId:     userId,
		}, nil
	}
	return nil, fmt.Errorf("invalid token")
}

func (j *jwtAuthService) DeleteAuth(givenUuid string) (int64, error) {
	deleted, err := j.client.Del(givenUuid).Result()
	if err != nil {
		return 0, err
	}
	return deleted, nil
}

func (j *jwtAuthService) ExtractToken(r *http.Request) string {
	bearToken := r.Header.Get("Authorization")
	//normally Authorization the_token_xxx
	strArr := strings.Split(bearToken, " ")
	if len(strArr) == 2 {
		return strArr[1]
	}
	return ""
}

func (j *jwtAuthService) TokenValid(tokenString string) error {
	token, err := j.VerifyToken(tokenString)
	if err != nil {
		return err
	}
	if _, ok := token.Claims.(jwt.Claims); !ok && !token.Valid {
		return err
	}
	return nil
}

func (j *jwtAuthService) VerifyRefreshToken(refreshToken string) (*jwt.Token, error) {
	token, err := jwt.Parse(refreshToken, func(token *jwt.Token) (interface{}, error) {
		//Make sure that the token method conform to "SigningMethodHMAC"
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(j.refreshKey), nil
	})
	if err != nil {
		return nil, err
	}
	if _, ok := token.Claims.(jwt.Claims); !ok && !token.Valid {
		return nil, fmt.Errorf("invald refresh token")
	}
	return token, nil
}

func (j *jwtAuthService) RefreshToken(token *jwt.Token) (uint64, *TokenDetails, error) {
	//Since token is valid, get the uuid:
	claims, ok := token.Claims.(jwt.MapClaims) //the token claims should conform to MapClaims
	if ok && token.Valid {
		refreshUuid, ok := claims["refresh_uuid"].(string) //convert the interface to string
		if !ok {
			return 0, nil, fmt.Errorf("failed to get refresh token id")
		}
		userId, err := strconv.ParseUint(fmt.Sprintf("%.f", claims["user_id"]), 10, 64)
		if err != nil {
			return 0, nil, err
		}
		//Delete the previous Refresh Token
		deleted, delErr := j.DeleteAuth(refreshUuid)
		if delErr != nil || deleted == 0 { //if any goes wrong
			return 0, nil, delErr
		}
		//Create new pairs of refresh and access tokens
		ts, err := j.CreateToken(userId)
		if err != nil {
			return 0, nil, err
		}
		return userId, ts, nil

	}
	return 0, nil, fmt.Errorf("refresh token expired")
}
