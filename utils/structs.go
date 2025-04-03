package utils

type User struct {
	UserID    string
	UserEmail string
	UserRole  int
}

type UserInfo struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	FullName string `json:"full_name"`
	Email    string `json:"email"`
	WalletID string `json:"wallet_id"`
	Role     int    `json:"role"`
}

// Define a reusable struct for credentials
type Credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type ConfigStruct struct {
	DatabaseURL         string `mapstructure:"DATABASE_URL"`
	DatabaseUsername    string `mapstructure:"DB_USERNAME"`
	DatabasePassword    string `mapstructure:"DB_PASSWORD"`
	EthereumRPC         string `mapstructure:"ETHEREUM_RPC"`
	JWTSecretKey        string `mapstructure:"JWT_SECRET"`
	JWTResetSecretKey   string `mapstructure:"JWT_RESET_SECRET"`
	SuperUserEmail      string `mapstructure:"SUPER_USER_EMAIL"`
	SuperUserPassword   string `mapstructure:"SUPER_USER_PASSWORD"`
	SendGridAPIKey      string `mapstructure:"SENDGRID_API_KEY"`
	SMTPHost            string `mapstructure:"SMTP_HOST"`
	SMTPPort            string `mapstructure:"SMTP_PORT"`
	SenderEmail         string `mapstructure:"SENDER_EMAIL"`
	SenderPassword      string `mapstructure:"SENDER_PASSWORD"`
	EtherscanAPI        string `mapstructure:"ETHERSCAN_API"`
	EtherscanAPIKey     string `mapstructure:"ETHERSCAN_API_KEY"`
	WalletEncryptionKey string `mapstructure:"WALLET_ENCRYPTION_KEY"`
}

const (
	ContentTypeJSON               = "application/json"
	ContentTypeHeader             = "Content-Type"
	ApplicationID                 = "application_id"
	RequestUserID                 = "user_id"
	UserEmail                     = "user_email"
	CtxUserID                     = "UserID"
	OfferID                       = "offer_id"
	DetailsID                     = "details_id"
	RepaymentID                   = "repayment_id"
	DisbursementID                = "disbursement_id"
	SettlementID                  = "settlement_id"
	StatusID                      = "status_id"
	StatusUpdateID                = "status_update_id"
	NotFoundID                    = "not_found_id"
	NotFoundStatus                = "not_found_status"
	NotFoundMessage               = "not_found_message"
	InvalidID                     = "invalid_id"
	InvalidStatus                 = "invalid_status"
	InvalidMessage                = "invalid_message"
	InternalServerError           = "internal_server_error"
	ServerErrorMessage            = "server_error_message"
	SuccessStatus                 = "success_status"
	SuccessMessage                = "success_message"
	ErrorStatus                   = "error_status"
	ErrorMessage                  = "error_message"
	WarningStatus                 = "warning_status"
	WarningMessage                = "warning_message"
	BadRequestStatus              = "bad_request_status"
	BadRequestMessage             = "bad_request_message"
	UnauthorizedStatus            = "unauthorized_status"
	UnauthorizedMessage           = "unauthorized_message"
	Status                        = "status"
	StatusOpen                    = "open"
	StatusClosed                  = "closed"
	StatusPending                 = "pending"
	StatusApproved                = "approved"
	StatusAccepted                = "Accepted"
	StatusRejected                = "rejected"
	StatusCancelled               = "cancelled"
	StatusCompleted               = "completed"
	StatusFailed                  = "failed"
	StatusExpired                 = "expired"
	StatusPaid                    = "paid"
	StatusUnpaid                  = "unpaid"
	StatusOverdue                 = "overdue"
	StatusDefault                 = "default"
	LoanID                        = "loan_id"
	LoanApplicationID             = "loan_application_id"
	LoanOfferID                   = "loan_offer_id"
	LoanDetailsID                 = "loan_details_id"
	LoanRepaymentID               = "loan_repayment_id"
	LoanDisbursementID            = "loan_disbursement_id"
	LoanSettlementID              = "loan_settlement_id"
	BorrowerID                    = "borrower_id"
	LenderID                      = "lender_id"
	Amount                        = "amount"
	InterestRate                  = "interest_rate"
	TermMonths                    = "term_months"
	Duration                      = "duration"
	TotalPayable                  = "total_payable"
	TotalPaid                     = "total_paid"
	TotalUnpaid                   = "total_unpaid"
	TotalOverdue                  = "total_overdue"
	TotalDefault                  = "total_default"
	TotalInterest                 = "total_interest"
	DocumentType                  = "document_type"
	DocumentNumber                = "document_number"
	KYCStatus                     = "kyc_status"
	KYCVerificationID             = "kyc_verification_id"
	KYCVerificationStatus         = "kyc_verification_status"
	KYCVerificationBy             = "kyc_verification_by"
	KYCVerificationAt             = "kyc_verification_at"
	KYCVerificationMessage        = "kyc_verification_message"
	KYCVerificationError          = "kyc_verification_error"
	KYCVerificationWarning        = "kyc_verification_warning"
	KYCVerificationSuccess        = "kyc_verification_success"
	KYCID                         = "kyc_id"
	Verified                      = "Verified"
	Unverified                    = "Unverified"
	ErrorTag                      = "error"
	ErrorMessageTag               = "error_message"
	WarningTag                    = "warning"
	WarningMessageTag             = "warning_message"
	SuccessTag                    = "success"
	SuccessMessageTag             = "success_message"
	BadRequestTag                 = "bad_request"
	BadRequestMessageTag          = "bad_request_message"
	UnauthorizedTag               = "unauthorized"
	UnauthorizedMessageTag        = "unauthorized_message"
	InternalServerErrorTag        = "internal_server_error"
	InternalServerErrorMessageTag = "internal_server_error_message"
	SenderEmail                   = "sender_email"
	ReceiverEmail                 = "receiver_email"
	FromTime                      = "fromtime"
	ToTime                        = "totime"
	RPCURLTag                     = "rpcURL"
	FromAddressTag                = "fromAddress"
	ToAddressTag                  = "toAddress"
	DerivedAddressTag             = "derivedAddress"
	AmountTag                     = "amount"
	PrivateKeyTag                 = "privateKey"
	PublicKeyTag                  = "publicKey"
	TransactionHashTag            = "transactionHash"
	TransactionStatusTag          = "transactionStatus"
	TransactionMessageTag         = "transactionMessage"
	TransactionErrorTag           = "transactionError"
	TransactionWarningTag         = "transactionWarning"
	TransactionSuccessTag         = "transactionSuccess"
	TransactionDetailsTag         = "transactionDetails"
	TransactionReceiptTag         = "transactionReceipt"
	TransactionBlockTag           = "transactionBlock"
	TransactionTimestampTag       = "transactionTimestamp"
	TransactionNonceTag           = "transactionNonce"
	TransactionGasTag             = "transactionGas"
	TransactionGasPriceTag        = "transactionGasPrice"
	TransactionDataTag            = "transactionData"
	TransactionInputTag           = "transactionInput"
	TransactionOutputTag          = "transactionOutput"
	TransactionTopicsTag          = "transactionTopics"
	TransactionVTag               = "transactionV"
	TransactionRTag               = "transactionR"
	RecoveredSenderTag            = "recoveredSender"
)
