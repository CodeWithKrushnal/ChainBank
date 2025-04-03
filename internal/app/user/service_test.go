package user

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/smtp"
	"testing"
	"time"

	EthMocks "github.com/CodeWithKrushnal/ChainBank/internal/app/ethereum/mocks"
	"github.com/CodeWithKrushnal/ChainBank/internal/repo"
	Repomocks "github.com/CodeWithKrushnal/ChainBank/internal/repo/mocks"
	"github.com/CodeWithKrushnal/ChainBank/utils"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"golang.org/x/crypto/bcrypt"
)

func TestGenerateLoginToken(t *testing.T) {
	ctx := context.Background()
	email := "test@example.com"
	originIP := "192.168.1.1"

	testCases := []struct {
		name          string
		config        utils.ConfigStruct
		expectError   bool
		validateToken bool
	}{
		{
			name: "Successful Token Generation",
			config: utils.ConfigStruct{
				JWTSecretKey: "supersecretkey",
			},
			expectError:   false,
			validateToken: true,
		},
		{
			name: "Invalid Secret Key",
			config: utils.ConfigStruct{
				JWTSecretKey: "",
			},
			expectError:   true,
			validateToken: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			token, err := GenerateLoginToken(ctx, email, originIP, tc.config)

			if tc.expectError {
				assert.Error(t, err)
				assert.Empty(t, token)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, token)

				if tc.validateToken {
					parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
						return []byte(tc.config.JWTSecretKey), nil
					})
					assert.NoError(t, err)
					assert.True(t, parsedToken.Valid)

					// Validate Expiration
					claims := jwt.MapClaims{}
					_, err = jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
						return []byte(tc.config.JWTSecretKey), nil
					})
					assert.NoError(t, err)

					exp, expOK := claims["exp"].(float64)
					iat, iatOK := claims["iat"].(float64)
					assert.True(t, expOK && iatOK)

					createdAt := time.Unix(int64(iat), 0)
					expirationTime := time.Unix(int64(exp), 0)
					assert.True(t, expirationTime.After(createdAt))
				}
			}
		})
	}
}

func TestGenerateResetToken(t *testing.T) {
	ctx := context.Background()
	email := "test@example.com"
	originIP := "192.168.1.1"

	testCases := []struct {
		name          string
		config        utils.ConfigStruct
		expectError   bool
		validateToken bool
	}{
		{
			name: "Successful Token Generation",
			config: utils.ConfigStruct{
				JWTResetSecretKey: "supersecretkey",
			},
			expectError:   false,
			validateToken: true,
		},
		{
			name: "Invalid Secret Key",
			config: utils.ConfigStruct{
				JWTResetSecretKey: "",
			},
			expectError:   true,
			validateToken: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			token, err := GenerateResetToken(ctx, email, originIP, tc.config)

			if tc.expectError {
				assert.Error(t, err)
				assert.Empty(t, token)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, token)

				if tc.validateToken {
					parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
						return []byte(tc.config.JWTResetSecretKey), nil
					})
					assert.NoError(t, err)
					assert.True(t, parsedToken.Valid)

					// Validate Expiration
					claims := jwt.MapClaims{}
					_, err = jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
						return []byte(tc.config.JWTResetSecretKey), nil
					})
					assert.NoError(t, err)

					exp, expOK := claims["exp"].(float64)
					iat, iatOK := claims["iat"].(float64)
					assert.True(t, expOK && iatOK)

					createdAt := time.Unix(int64(iat), 0)
					expirationTime := time.Unix(int64(exp), 0)
					assert.True(t, expirationTime.After(createdAt))

					assert.True(t, claims["reset"] == true)
				}
			}
		})
	}
}

func TestPrivateKeyToHex(t *testing.T) {
	testCases := []struct {
		name        string
		privateKey  *ecdsa.PrivateKey
		expectError bool
	}{
		{
			name: "Valid Private Key",
			privateKey: func() *ecdsa.PrivateKey {
				key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
				return key
			}(),
			expectError: false,
		},
		{
			name:        "Nil Private Key",
			privateKey:  nil,
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hexStr, err := PrivateKeyToHex(tc.privateKey)

			if tc.expectError {
				assert.Error(t, err)
				assert.Empty(t, hexStr)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, hexStr)

				// Convert back to bytes and check length consistency
				privateKeyBytes := crypto.FromECDSA(tc.privateKey)
				expectedHex := hex.EncodeToString(privateKeyBytes)
				assert.Equal(t, expectedHex, hexStr)
			}
		})
	}

}

type UserServiceTestSuite struct {
	suite.Suite
	userRepo      *Repomocks.UserStorer
	walletRepo    *Repomocks.WalletStorer
	ethRepo       *EthMocks.EthRepo
	configDetails utils.ConfigStruct
	service       Service
}

func TestUserServiceTestSuite(t *testing.T) {
	suite.Run(t, new(UserServiceTestSuite))
}

func (suite *UserServiceTestSuite) SetupTest() {
	suite.userRepo = &Repomocks.UserStorer{}
	suite.walletRepo = &Repomocks.WalletStorer{}
	suite.ethRepo = &EthMocks.EthRepo{}
	suite.configDetails = utils.ConfigStruct{
		JWTSecretKey:   "THISISJWTSECRET",
		SMTPHost:       "smtp.example.com",
		SMTPPort:       "587",
		SenderEmail:    "sender@example.com",
		SenderPassword: "password"}
	suite.service = NewService(context.Background(), suite.userRepo, suite.walletRepo, suite.ethRepo, suite.configDetails)
}

func (suite *UserServiceTestSuite) TearDownTest() {
	suite.userRepo.AssertExpectations(suite.T())
	suite.walletRepo.AssertExpectations(suite.T())
	suite.ethRepo.AssertExpectations(suite.T())
}

// MockSMTP is a mock implementation of smtp.SendMail function
type MockSMTP struct {
	mock.Mock
}

func (m *MockSMTP) SendMail(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
	args := m.Called(addr, a, from, to, msg)
	return args.Error(0)
}

func (suite *UserServiceTestSuite) TestAuthenticateUser() {
	ctx := context.Background()
	originIP := "192.168.1.1"

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("ValidPassword"), bcrypt.DefaultCost)

	tests := []struct {
		name           string
		credentials    utils.Credentials
		mockSetup      func()
		expectedErr    error
		expectedTokens bool
	}{
		{
			name: "Valid credentials",
			credentials: utils.Credentials{
				Email:    "test@example.com",
				Password: "ValidPassword",
			},
			mockSetup: func() {
				suite.userRepo.On("GetUserByEmail", ctx, "test@example.com").
					Return(repo.User{Email: "test@example.com", Password: string(hashedPassword)}, nil).Once()
			},
			expectedErr:    nil,
			expectedTokens: true,
		},
		{
			name: "User not found",
			credentials: utils.Credentials{
				Email:    "unknown@example.com",
				Password: "ValidPassword",
			},
			mockSetup: func() {
				suite.userRepo.On("GetUserByEmail", ctx, "unknown@example.com").
					Return(repo.User{}, errors.New("user not found")).Once()
			},
			expectedErr:    fmt.Errorf(utils.ErrorFormat, utils.ErrUserNotFound, errors.New("user not found")),
			expectedTokens: false,
		},
		{
			name: "Incorrect password",
			credentials: utils.Credentials{
				Email:    "test@example.com",
				Password: "WrongPassword",
			},
			mockSetup: func() {
				suite.userRepo.On("GetUserByEmail", ctx, "test@example.com").
					Return(repo.User{Email: "test@example.com", Password: string(hashedPassword)}, nil).Once()
			},
			expectedErr:    fmt.Errorf(utils.ErrorFormat, utils.ErrInvalidCredentials, bcrypt.ErrMismatchedHashAndPassword),
			expectedTokens: false,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			tt.mockSetup()

			tokens, err := suite.service.AuthenticateUser(ctx, tt.credentials, originIP)

			suite.Equal(tt.expectedErr, err)
			suite.Equal(tt.expectedTokens, tokens != nil)
		})
	}

	suite.TearDownTest()
}

func (suite *UserServiceTestSuite) TestInsertKYCVerificationService() {
	ctx := context.Background()

	tests := []struct {
		name               string
		userEmail          string
		documentType       string
		documentNumber     string
		verificationStatus string
		mockSetup          func()
		expectedKYCID      string
		expectedErr        error
	}{
		{
			name:               "Successful KYC insertion",
			userEmail:          "test@example.com",
			documentType:       "Passport",
			documentNumber:     "A1234567",
			verificationStatus: "Verified",
			mockSetup: func() {
				suite.userRepo.On("GetUserByEmail", ctx, "test@example.com").
					Return(repo.User{ID: "user123", Email: "test@example.com"}, nil).Once()

				suite.userRepo.On("InsertKYCVerification", ctx, "user123", "Passport", "A1234567", "Verified").
					Return("kyc123", nil).Once()
			},
			expectedKYCID: "kyc123",
			expectedErr:   nil,
		},
		{
			name:               "User not found",
			userEmail:          "unknown@example.com",
			documentType:       "Passport",
			documentNumber:     "A1234567",
			verificationStatus: "Verified",
			mockSetup: func() {
				suite.userRepo.On("GetUserByEmail", ctx, "unknown@example.com").
					Return(repo.User{}, errors.New("user not found")).Once()
			},
			expectedKYCID: "",
			expectedErr:   fmt.Errorf(utils.ErrorFormat, utils.ErrUserNotFound, errors.New("user not found")),
		},
		{
			name:               "KYC insertion failure",
			userEmail:          "test@example.com",
			documentType:       "Driver's License",
			documentNumber:     "DL987654",
			verificationStatus: "Pending",
			mockSetup: func() {
				suite.userRepo.On("GetUserByEmail", ctx, "test@example.com").
					Return(repo.User{ID: "user123", Email: "test@example.com"}, nil).Once()

				suite.userRepo.On("InsertKYCVerification", ctx, "user123", "Driver's License", "DL987654", "Pending").
					Return("", errors.New("database error")).Once()
			},
			expectedKYCID: "",
			expectedErr:   fmt.Errorf(utils.ErrorFormat, utils.ErrKYCVerificationInsertion, errors.New("database error")),
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			tt.mockSetup()

			kycID, err := suite.service.InsertKYCVerificationService(ctx, tt.userEmail, tt.documentType, tt.documentNumber, tt.verificationStatus)

			suite.Equal(tt.expectedKYCID, kycID)
			suite.Equal(tt.expectedErr, err)
		})
	}
}

func (suite *UserServiceTestSuite) TestGetAllKYCVerificationsService() {
	ctx := context.Background()

	tests := []struct {
		name           string
		mockSetup      func()
		expectedResult []repo.KYCRecord
		expectedErr    error
	}{
		{
			name: "Successfully retrieve all KYC records",
			mockSetup: func() {
				mockRecords := []repo.KYCRecord{
					{KYCID: "kyc1", UserID: "user1", DocumentType: "Passport", DocumentNumber: "A1234567", VerificationStatus: "Verified"},
					{KYCID: "kyc2", UserID: "user2", DocumentType: "Driver's License", DocumentNumber: "DL987654", VerificationStatus: "Pending"},
				}

				suite.userRepo.On("GetAllKYCVerifications", ctx).
					Return(mockRecords, nil).Once()
			},
			expectedResult: []repo.KYCRecord{
				{KYCID: "kyc1", UserID: "user1", DocumentType: "Passport", DocumentNumber: "A1234567", VerificationStatus: "Verified"},
				{KYCID: "kyc2", UserID: "user2", DocumentType: "Driver's License", DocumentNumber: "DL987654", VerificationStatus: "Pending"},
			},
			expectedErr: nil,
		},
		{
			name: "Database error while fetching KYC records",
			mockSetup: func() {
				suite.userRepo.On("GetAllKYCVerifications", ctx).
					Return(nil, errors.New("database error")).Once()
			},
			expectedResult: nil,
			expectedErr:    fmt.Errorf(utils.ErrorFormat, utils.ErrFetchKYCDetailedInfo, errors.New("database error")),
		},
		{
			name: "No KYC records found",
			mockSetup: func() {
				suite.userRepo.On("GetAllKYCVerifications", ctx).
					Return([]repo.KYCRecord{}, nil).Once()
			},
			expectedResult: []repo.KYCRecord{},
			expectedErr:    nil,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			tt.mockSetup()

			result, err := suite.service.GetAllKYCVerificationsService(ctx)

			suite.Equal(tt.expectedResult, result)
			suite.Equal(tt.expectedErr, err)
		})
	}
}

func (suite *UserServiceTestSuite) TestUpdateKYCVerificationStatusService() {
	ctx := context.Background()
	validKYCID := "kyc123"
	invalidKYCID := "invalid_kyc"
	verificationStatus := "Verified"
	verifiedBy := "admin@example.com"

	tests := []struct {
		name        string
		kycID       string
		mockSetup   func()
		expectedErr error
	}{
		{
			name:  "Successfully update KYC verification status",
			kycID: validKYCID,
			mockSetup: func() {
				suite.userRepo.On("UpdateKYCVerificationStatus", ctx, validKYCID, verificationStatus, verifiedBy).
					Return(nil).Once()
			},
			expectedErr: nil,
		},
		{
			name:  "Database error while updating KYC status",
			kycID: validKYCID,
			mockSetup: func() {
				suite.userRepo.On("UpdateKYCVerificationStatus", ctx, validKYCID, verificationStatus, verifiedBy).
					Return(errors.New("database error")).Once()
			},
			expectedErr: fmt.Errorf(utils.ErrorFormat, utils.ErrUpdatingKYCVerificationStatus, errors.New("database error")),
		},
		{
			name:  "Invalid KYC ID",
			kycID: invalidKYCID, // Use the correct invalid KYC ID here
			mockSetup: func() {
				suite.userRepo.On("UpdateKYCVerificationStatus", ctx, invalidKYCID, verificationStatus, verifiedBy).
					Return(errors.New("KYC ID not found")).Once()
			},
			expectedErr: fmt.Errorf(utils.ErrorFormat, utils.ErrUpdatingKYCVerificationStatus, errors.New("KYC ID not found")),
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			tt.mockSetup()

			err := suite.service.UpdateKYCVerificationStatusService(ctx, tt.kycID, verificationStatus, verifiedBy)

			suite.Equal(tt.expectedErr, err)
			suite.userRepo.AssertExpectations(suite.T()) // Ensures all mock expectations were met
		})
	}
}

func (suite *UserServiceTestSuite) TestGetKYCDetailedInfo() {
	ctx := context.Background()

	suite.Run("Fetch KYC details by valid KYC ID", func() {
		mockKYCRecords := []repo.KYCRecord{{KYCID: "kyc-123", UserID: "user-123"}}
		suite.userRepo.On("GetKYCDetailedInfo", ctx, "kyc-123", "").Return(mockKYCRecords, nil).Once()

		result, err := suite.service.GetKYCDetailedInfo(ctx, "kyc-123", "")
		suite.NoError(err)
		suite.Equal(mockKYCRecords, result)
	})

	suite.Run("Fetch KYC details by valid User Email", func() {
		mockUser := repo.User{ID: "user-123", Email: "test@example.com"}
		mockKYCRecords := []repo.KYCRecord{{KYCID: "kyc-123", UserID: "user-123"}}

		suite.userRepo.On("GetUserByEmail", ctx, "test@example.com").Return(mockUser, nil).Once()
		suite.userRepo.On("GetKYCDetailedInfo", ctx, "", "user-123").Return(mockKYCRecords, nil).Once()

		result, err := suite.service.GetKYCDetailedInfo(ctx, "", "test@example.com")
		suite.NoError(err)
		suite.Equal(mockKYCRecords, result)
	})

	suite.Run("Error: Both KYC ID and User Email provided", func() {
		result, err := suite.service.GetKYCDetailedInfo(ctx, "kyc-123", "test@example.com")
		suite.Error(err)
		suite.Nil(result)
	})

	suite.Run("Error: Both KYC ID and User Email are empty", func() {
		result, err := suite.service.GetKYCDetailedInfo(ctx, "", "")
		suite.Error(err)
		suite.Nil(result)
	})

	suite.Run("Error: User not found", func() {
		suite.userRepo.On("GetUserByEmail", ctx, "non-existent@example.com").Return(repo.User{}, errors.New("user not found")).Once()

		result, err := suite.service.GetKYCDetailedInfo(ctx, "", "non-existent@example.com")
		suite.Error(err)
		suite.Nil(result)
	})

	suite.Run("Error: KYC record fetch failure", func() {
		suite.userRepo.On("GetKYCDetailedInfo", ctx, "kyc-123", "").Return(nil, errors.New("fetch error")).Once()

		result, err := suite.service.GetKYCDetailedInfo(ctx, "kyc-123", "")
		suite.Error(err)
		suite.Nil(result)
	})
}

func (suite *UserServiceTestSuite) TestGetUserByID() {
	tests := []struct {
		name          string
		userID        string
		mockUser      repo.User
		mockRole      int
		mockUserErr   error
		mockRoleErr   error
		expectedUser  utils.User
		expectedError string
	}{
		{
			name:   "Success - Valid User",
			userID: "123",
			mockUser: repo.User{
				ID:    "123",
				Email: "test@example.com",
			},
			mockRole: 3,
			expectedUser: utils.User{
				UserID:    "123",
				UserEmail: "test@example.com",
				UserRole:  3,
			},
			expectedError: "",
		},
		{
			name:          "Failure - User Not Found",
			userID:        "999",
			mockUserErr:   fmt.Errorf("user not found"),
			expectedUser:  utils.User{},
			expectedError: "user not found",
		},
		{
			name:   "Failure - Role Fetch Error",
			userID: "123",
			mockUser: repo.User{
				ID:    "123",
				Email: "test@example.com",
			},
			mockRoleErr:   fmt.Errorf("role fetch error"),
			expectedUser:  utils.User{},
			expectedError: "role fetch error",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			suite.userRepo.On("GetuserByID", mock.Anything, tt.userID).
				Return(tt.mockUser, tt.mockUserErr).Once()
			if tt.mockUserErr == nil {
				suite.userRepo.On("GetUserHighestRole", mock.Anything, tt.userID).
					Return(tt.mockRole, tt.mockRoleErr).Once()
			}

			user, err := suite.service.GetUserByID(context.Background(), tt.userID)

			if tt.expectedError == "" {
				suite.NoError(err)
				suite.Equal(tt.expectedUser, user)
			} else {
				suite.Error(err)
				suite.Contains(err.Error(), tt.expectedError)
			}

			suite.userRepo.AssertExpectations(suite.T())
		})
	}
}

func (suite *UserServiceTestSuite) TestGetUserByEmail() {
    ctx := context.Background()

    testCases := []struct {
        name        string
        email       string
        mockReturn  repo.User
        mockError   error
        expectError bool
        expectedErr string
    }{
        {
            name: "Success - User Found",
            email: "john@example.com",
            mockReturn: repo.User{
                ID:       "123",
                Username:     "John Doe",
                Email:    "john@example.com",
                Password: "hashedpassword",
            },
            mockError:   nil,
            expectError: false,
        },
        {
            name:        "Failure - User Not Found",
            email:       "notfound@example.com",
            mockReturn:  repo.User{},
            mockError:   errors.New("user not found"),
            expectError: true,
            expectedErr: "user not found",
        },
        {
            name:        "Failure - Database Error",
            email:       "dbfail@example.com",
            mockReturn:  repo.User{},
            mockError:   errors.New("database connection error"),
            expectError: true,
            expectedErr: "database connection error",
        },
    }

    for _, tc := range testCases {
        suite.Run(tc.name, func() {
            // Mock behavior
            suite.userRepo.On("GetUserByEmail", ctx, tc.email).Return(tc.mockReturn, tc.mockError).Once()

            // Call function
            user, err := suite.service.GetUserByEmail(ctx, tc.email)

            // Assertions
            if tc.expectError {
                suite.NotNil(err)
                suite.EqualError(err, tc.expectedErr)
            } else {
                suite.Nil(err)
                suite.Equal(tc.mockReturn, user)
            }

            suite.userRepo.AssertExpectations(suite.T())
        })
    }
}

func (suite *UserServiceTestSuite) TestGenerateEmailVerificationToken() {
    ctx := context.Background()
    testCases := []struct {
        name         string
        email        string
        originIP     string
        jwtSecret    string
        expectError  bool
        expectedErr  string
    }{
        {
            name:        "Success - Token Generated",
            email:       "user@example.com",
            originIP:    "192.168.1.1",
            jwtSecret:   "THISISJWTSECRET", // Valid secret
            expectError: false,
        },
    }

    for _, tc := range testCases {
        suite.Run(tc.name, func() {
            suite.configDetails.JWTSecretKey = tc.jwtSecret

            // Call function
            token, err := suite.service.GenerateEmailVerificationToken(ctx, tc.email, tc.originIP)

            // Assertions
            if tc.expectError {
                suite.NotNil(err)
                suite.Contains(err.Error(), tc.expectedErr)
                suite.Empty(token)
            } else {
                suite.Nil(err)
                suite.NotEmpty(token)

                // Validate the token structure
                parsedToken, parseErr := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
                    return []byte(suite.configDetails.JWTSecretKey), nil
                })

                suite.Nil(parseErr)
                suite.NotNil(parsedToken)
                suite.True(parsedToken.Valid)

                // Validate token claims
                claims, ok := parsedToken.Claims.(jwt.MapClaims)
                suite.True(ok)
                suite.Equal(tc.email, claims["email"])
                suite.Equal(tc.originIP, claims["origin"])
                suite.True(claims["verify"].(bool))
            }
        })
    }
}

func (suite *UserServiceTestSuite) TestValidateEmailVerificationToken() {
    validEmail := "user@example.com"
    validOriginIP := "192.168.1.1"
    jwtSecret := suite.configDetails.JWTSecretKey

    // Helper function to generate test tokens
    generateTestToken := func(claims jwt.MapClaims, secret string, signingMethod jwt.SigningMethod) string {
        token := jwt.NewWithClaims(signingMethod, claims)
        tokenString, _ := token.SignedString([]byte(secret))
        return tokenString
    }

    // Generate a valid token
    validClaims := jwt.MapClaims{
        "email":  validEmail,
        "exp":    time.Now().Add(time.Minute * 5).Unix(), // Not expired
        "iat":    time.Now().Unix(),
        "origin": validOriginIP,
        "verify": true,
    }
    validToken := generateTestToken(validClaims, jwtSecret, jwt.SigningMethodHS256)

    // Generate an expired token
    expiredClaims := jwt.MapClaims{
        "email":  validEmail,
        "exp":    time.Now().Add(-time.Minute * 5).Unix(), // Already expired
        "iat":    time.Now().Add(-time.Minute * 10).Unix(),
        "origin": validOriginIP,
        "verify": true,
    }
    expiredToken := generateTestToken(expiredClaims, jwtSecret, jwt.SigningMethodHS256)

    // Generate a token with missing/invalid "verify" claim
    missingVerifyClaims := jwt.MapClaims{
        "email":  validEmail,
        "exp":    time.Now().Add(time.Minute * 5).Unix(),
        "iat":    time.Now().Unix(),
        "origin": validOriginIP,
        // "verify" is missing
    }
    missingVerifyToken := generateTestToken(missingVerifyClaims, jwtSecret, jwt.SigningMethodHS256)

    testCases := []struct {
        name        string
        tokenString string
        expectError bool
        expectedErr string
    }{
        {
            name:        "Success - Valid Token",
            tokenString: validToken,
            expectError: false,
        },
        {
            name:        "Failure - Expired Token",
            tokenString: expiredToken,
            expectError: true,
            expectedErr: "token is expired",
        },
        {
            name:        "Failure - Missing Verify Claim",
            tokenString: missingVerifyToken,
            expectError: true,
            expectedErr: "invalid verification token",
        },
        {
            name:        "Failure - Malformed Token",
            tokenString: "random-invalid-token",
            expectError: true,
            expectedErr: "token contains an invalid number of segments",
        },
    }

    for _, tc := range testCases {
        suite.Run(tc.name, func() {
            claims, err := suite.service.ValidateEmailVerificationToken(tc.tokenString)

            if tc.expectError {
                suite.NotNil(err)
                suite.Contains(err.Error(), tc.expectedErr)
                suite.Nil(claims)
            } else {
                suite.Nil(err)
                suite.NotNil(claims)
                suite.Equal(validEmail, claims["email"])
                suite.Equal(validOriginIP, claims["origin"])
                suite.True(claims["verify"].(bool))
            }
        })
    }
}

func (suite *UserServiceTestSuite) TestGetRequestLogs() {
    ctx := context.Background()

    sampleLogs := []repo.RequestLog{
        {
            RequestID:      "req-123",
            UserID:         "user-1",
            Endpoint:       "/api/login",
            HTTPMethod:     "POST",
            RequestPayload: []byte(`{"username": "test"}`),
            ResponseStatus: 200,
            ResponseTimeMs: 150,
            IPAddress:      "192.168.1.10",
            CreatedAt:      time.Now(),
        },
        {
            RequestID:      "req-124",
            UserID:         "user-2",
            Endpoint:       "/api/register",
            HTTPMethod:     "POST",
            RequestPayload: []byte(`{"email": "test@example.com"}`),
            ResponseStatus: 201,
            ResponseTimeMs: 200,
            IPAddress:      "192.168.1.20",
            CreatedAt:      time.Now(),
        },
    }

    testCases := []struct {
        name         string
        filter       repo.RequestLogFilter
        mockResponse []repo.RequestLog
        mockError    error
        expectError  bool
        expectedLogs []repo.RequestLog
    }{
        {
            name:         "Success - Logs Found",
            filter:       repo.RequestLogFilter{UserID: "user-1"},
            mockResponse: sampleLogs[:1], // Return first log
            mockError:    nil,
            expectError:  false,
            expectedLogs: sampleLogs[:1],
        },
        {
            name:         "Failure - No Logs Found",
            filter:       repo.RequestLogFilter{UserID: "user-99"},
            mockResponse: []repo.RequestLog{}, // Empty result
            mockError:    nil,
            expectError:  false,
            expectedLogs: []repo.RequestLog{},
        },
        {
            name:         "Failure - Repository Error",
            filter:       repo.RequestLogFilter{UserID: "user-1"},
            mockResponse: nil,
            mockError:    errors.New("database connection failed"),
            expectError:  true,
            expectedLogs: nil,
        },
    }

    for _, tc := range testCases {
        suite.Run(tc.name, func() {
            // Mock repository call
            suite.userRepo.On("GetRequestLogs", ctx, tc.filter).Return(tc.mockResponse, tc.mockError).Once()

            // Execute function
            logs, err := suite.service.GetRequestLogs(ctx, tc.filter)

            if tc.expectError {
                suite.NotNil(err)
                suite.Contains(err.Error(), tc.mockError.Error())
                suite.Nil(logs)
            } else {
                suite.Nil(err)
                suite.Equal(tc.expectedLogs, logs)
            }

            suite.userRepo.AssertExpectations(suite.T())
        })
    }
}

func (suite *UserServiceTestSuite) TestGetRequestLogStats() {
    ctx := context.Background()
    now := time.Now()
    earlier := now.Add(-24 * time.Hour)

    testCases := []struct {
        name         string
        filter       repo.RequestLogStatsFilter
        mockResponse interface{}
        mockError    error
        expectError  bool
        expectedData interface{}
    }{
        {
            name: "Success - Count Request Logs",
            filter: repo.RequestLogStatsFilter{
                FromTime: &earlier,
                ToTime:   &now,
                Column:   "request_id",
                Count:    true,
            },
            mockResponse: 100, // Example count result
            mockError:    nil,
            expectError:  false,
            expectedData: 100,
        },
        {
            name: "Success - Average Response Time",
            filter: repo.RequestLogStatsFilter{
                FromTime: &earlier,
                ToTime:   &now,
                Column:   "response_time_ms",
                Avg:      true,
            },
            mockResponse: 250.5, // Example average response time
            mockError:    nil,
            expectError:  false,
            expectedData: 250.5,
        },
        {
            name: "Success - Unique IPs",
            filter: repo.RequestLogStatsFilter{
                FromTime: &earlier,
                ToTime:   &now,
                Column:   "ip_address",
                Unique:   true,
            },
            mockResponse: 5, // Example unique IP count
            mockError:    nil,
            expectError:  false,
            expectedData: 5,
        },
        {
            name: "Failure - Repository Error",
            filter: repo.RequestLogStatsFilter{
                FromTime: &earlier,
                ToTime:   &now,
                Column:   "request_id",
                Count:    true,
            },
            mockResponse: nil,
            mockError:    errors.New("database query failed"),
            expectError:  true,
            expectedData: nil,
        },
    }

    for _, tc := range testCases {
        suite.Run(tc.name, func() {
            // Mock repository call
            suite.userRepo.On("GetRequestLogStats", ctx, tc.filter).Return(tc.mockResponse, tc.mockError).Once()

            // Execute function
            data, err := suite.service.GetRequestLogStats(ctx, tc.filter)

            if tc.expectError {
                suite.NotNil(err)
                suite.Contains(err.Error(), tc.mockError.Error())
                suite.Nil(data)
            } else {
                suite.Nil(err)
                suite.Equal(tc.expectedData, data)
            }

            suite.userRepo.AssertExpectations(suite.T())
        })
    }
}

func (suite *UserServiceTestSuite) TestGetUserInfo() {
    ctx := context.Background()

    testCases := []struct {
        name         string
        userID       string
        mockResponse utils.UserInfo
        mockError    error
        expectError  bool
        expectedData utils.UserInfo
    }{
        {
            name:   "Success - Valid User ID",
            userID: "12345",
            mockResponse: utils.UserInfo{
                UserID:   "12345",
                Username: "johndoe",
                FullName: "John Doe",
                Email:    "john@example.com",
                WalletID: "wallet_001",
                Role:     1,
            },
            mockError:   nil,
            expectError: false,
            expectedData: utils.UserInfo{
                UserID:   "12345",
                Username: "johndoe",
                FullName: "John Doe",
                Email:    "john@example.com",
                WalletID: "wallet_001",
                Role:     1,
            },
        },
        {
            name:         "Failure - User Not Found",
            userID:       "99999",
            mockResponse: utils.UserInfo{},
            mockError:    errors.New("user not found"),
            expectError:  true,
            expectedData: utils.UserInfo{},
        },
        {
            name:         "Failure - Database Error",
            userID:       "12345",
            mockResponse: utils.UserInfo{},
            mockError:    errors.New("database query failed"),
            expectError:  true,
            expectedData: utils.UserInfo{},
        },
    }

    for _, tc := range testCases {
        suite.Run(tc.name, func() {
            // Mock repository call
            suite.userRepo.On("GetUserInfo", ctx, tc.userID).Return(tc.mockResponse, tc.mockError).Once()

            // Execute function
            data, err := suite.service.GetUserInfo(ctx, tc.userID)

            if tc.expectError {
                suite.NotNil(err)
                suite.Contains(err.Error(), tc.mockError.Error())
                suite.Equal(tc.expectedData, data)
            } else {
                suite.Nil(err)
                suite.Equal(tc.expectedData, data)
            }

            suite.userRepo.AssertExpectations(suite.T())
        })
    }
}

func (suite *UserServiceTestSuite) TestGetTransactionStats() {
    ctx := context.Background()

    testCases := []struct {
        name          string
        filter        repo.TransactionStatsFilter
        mockResponse  interface{}
        mockError     error
        expectError   bool
        expectedData  interface{}
    }{
        {
            name: "Success - Valid Filter",
            filter: repo.TransactionStatsFilter{
                FromTime: timePtr(time.Now().Add(-24 * time.Hour)),
                ToTime:   timePtr(time.Now()),
                Column:   "amount",
                Sum:      true,
            },
            mockResponse: map[string]interface{}{
                "sum": 10000.0,
            },
            mockError:    nil,
            expectError:  false,
            expectedData: map[string]interface{}{
                "sum": 10000.0,
            },
        },
        {
            name: "Failure - No Transactions Found",
            filter: repo.TransactionStatsFilter{
                FromTime: timePtr(time.Now().Add(-24 * time.Hour)),
                ToTime:   timePtr(time.Now()),
                Column:   "amount",
                Count:    true,
            },
            mockResponse: map[string]interface{}{},
            mockError:    nil,
            expectError:  false,
            expectedData: map[string]interface{}{},
        },
        {
            name: "Failure - Database Error",
            filter: repo.TransactionStatsFilter{
                FromTime: timePtr(time.Now().Add(-24 * time.Hour)),
                ToTime:   timePtr(time.Now()),
                Column:   "amount",
                Avg:      true,
            },
            mockResponse: nil,
            mockError:    errors.New("database query failed"),
            expectError:  true,
            expectedData: nil,
        },
    }

    for _, tc := range testCases {
        suite.Run(tc.name, func() {
            // Mock repository call
            suite.userRepo.On("GetTransactionStats", ctx, tc.filter).Return(tc.mockResponse, tc.mockError).Once()

            // Execute function
            data, err := suite.service.GetTransactionStats(ctx, tc.filter)

            if tc.expectError {
                suite.NotNil(err)
                suite.Contains(err.Error(), tc.mockError.Error())
                suite.Equal(tc.expectedData, data)
            } else {
                suite.Nil(err)
                suite.Equal(tc.expectedData, data)
            }

            suite.userRepo.AssertExpectations(suite.T())
        })
    }
}

func timePtr(t time.Time) *time.Time {
    return &t
}

func (suite *UserServiceTestSuite) TestGetConfig() {
    ctx := context.Background()

    testCases := []struct {
        name         string
        expectedData utils.ConfigStruct
        expectError  bool
    }{
        {
            name: "Success - Valid Config",
            expectedData: suite.configDetails, 
            expectError:  false,
        },
    }

    for _, tc := range testCases {
        suite.Run(tc.name, func() {
            // Execute function
            config, err := suite.service.GetConfig(ctx)

            // Validations
            suite.Nil(err)
            suite.Equal(tc.expectedData, config)
        })
    }
}