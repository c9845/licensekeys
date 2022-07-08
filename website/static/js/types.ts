/**
 * types.ts
 * The code in this file defines the customer types used for this app. These
 * closely match the types defined for the database schema.
 * 
 * The order of these interfaces matches the order of the db-*.go files.
 */

interface activityLog {
    ID:                 number,
    DatetimeCreated:    string,
    TimestampCreated:   number, //unix timestamp in nanoseconds
    Method:             string, //GET, POST
    URL:                string, //the endpoint accessed
    RemoteIP:           string,
    UserAgent:          string,
    TimeDuration:       number, //milliseconds it took for server to complete the request
    PostFormValues:     string, //json encoded form values passed in request
    UserID:             number, 
    APIKeyID:           number,

    //JOINed fields
    Username:           string,
    APIKeyDescription:  string,
    APIKeyK:            string,
}

interface apiKey{
    ID:                 number,
    DatetimeCreated:    string,
    DatetimeModified:   string,
    CreatedByUserID:    number,
    Active:             boolean,

    Description:        string, //so user can identify what the api key is used for
    K:                  string, //the actual api key
    
    //JOINed fields
    CreatedByUsername:  string,
}

interface app {
    ID:                 number,
    DatetimeCreated:    string,
    DatetimeModified:   string,
    CreatedByUserID:    number,
    Active:             boolean,

    Name:               string,
	DaysToExpiration:   number,  //the number of days to add on to "today" to calculate a default expiration date of a license
	FileFormat:         string,  //yaml, json, etc.
    ShowLicenseID:      boolean, //if the ID field of a created license file will be populated/non-zero.
	ShowAppName:        boolean, //if the Application field of a created license file will be populated/non-blank.
    DownloadFilename:   string,
}

//This must match the formats defined in keyfile-fileFormats.go.
const fileFormatYAML: string =    "yaml";
const fileFormatJSON: string =    "json";
const fileFormats: string[] =     [
    fileFormatYAML, 
    fileFormatJSON,
];

interface appSettings {
    ID:                     number,
    DatetimeModified:       string,

    EnableActivityLogging:  boolean, //whether or not the app tracks user activity (page views, endpoints, etc.)
    AllowAPIAccess:         boolean, //whether or not external access to this app is allowed
    Allow2FactorAuth:       boolean, //if 2 factor authentication can be used
    Force2FactorAuth:       boolean, //if all users are required to have 2 factor auth enabled prior to logging in (check if at least one user has 2fa enabled first to prevent lock out!)
    ForceSingleSession:     boolean, //user can only be logged into the app in one browser at a time. used as a security tool.
}

interface customFieldDefined {
    ID:                 number,
    DatetimeCreated:    string,
    DatetimeModified:   string,
    CreatedByUserID:    number,
    Active:             boolean,

    AppID:              number,
    Type:               string,
    Name:               string,
    Instructions:       string, //what field is for and expected value

    IntegerDefaultValue:        number,
    DecimalDefaultValue:        number,
    TextDefaultValue:           string,
    BoolDefaultValue:           boolean,
    MultiChoiceDefaultValue:    string,
    DateDefaultIncrement:       number, //number of days incremented from "today", date license is being created

    NumberMinValue:     number,
    NumberMaxValue:     number,
    MultiChoiceOptions: string,

    //When saving a license, we retrieve the defined fields for an app 
    //and set the value for each field using the same list of objects 
    //returned just for ease of use and not changing types. Therefore,
    //we need fields in this type/object to store the chosen/provided
    //values for each field. Only one of these fields is populated per
    //the Type field.
    IntegerValue:       number,
    DecimalValue:       number,
    TextValue:          string,
    BoolValue:          boolean,
    MultiChoiceValue:   string,
    DateValue:          string,
}

const customFieldTypeInteger: string =        "Integer";
const customFieldTypeDecimal: string =        "Decimal";
const customFieldTypeText: string =           "Text";
const customFieldTypeBoolean: string =        "Boolean";
const customFieldTypeMultiChoice: string =    "Multi-Choice";
const customFieldTypeDate: string =           "Date";
const customFieldTypes: string[] = [
    customFieldTypeInteger, 
    customFieldTypeDecimal, 
    customFieldTypeText, 
    customFieldTypeMultiChoice,
    customFieldTypeBoolean,
    customFieldTypeDate,
];

interface customFieldResults{
    ID:                 number,
    DatetimeCreated:    string,
    Active:             boolean,
    
    CreatedByUserID:    number,
    CreatedByAPIKeyID:  number,

    LicenseID:              number, //what license this result is for
    CustomFieldDefinedID:   number, //just for reference, although we will probably never need to refer back to defined field once license is created.
    CustomFieldType:        string, //integer, decimal, text, etc.
    CustomFieldName:        string, //so that if defined field's name changes, we still have name as was set when license was created.
    
    //only one of these has a valid value, per CustomFieldType
    IntegerValue:       number,
    DecimalValue:       number,
    TextValue:          string,
    BoolValue:          boolean,
    MultiChoiceValue:   string,
    DateValue:          string,
}

interface keyPair {
    ID:                 number,
    DatetimeCreated:    string,
    DatetimeModified:   string,
    CreatedByUserID:    number,
    Active:             boolean,

    AppID:                  number,
    Name:                   string,
    PrivateKey:             string, //should NEVER be returned
    PublicKey:              string, //returned so user can copy-paste into their app's code
    AlgorithmType:          string, //ecdsa, rsa, etc.
    PrivateKeyEncrypted:    boolean, //whether the private key is stored in plaintext or encrypted
    IsDefault:              boolean, //if this is the default keypair for the app
}

const keyPairAlgoECDSAP256: string =    "ECDSA (P256)";
const keyPairAlgoECDSAP384: string =    "ECDSA (P384)";
const keyPairAlgoECDSAP521: string =    "ECDSA (P521)";
const keyPairAlgoRSA2048: string =      "RSA (2048-bit)";
const keyPairAlgoRSA4096: string =      "RSA (4096-bit)";
const keyPairAlgoED25519: string =      "ED25519";
const keyPairAlgoTypes: string[] = [
    keyPairAlgoECDSAP256, 
    keyPairAlgoECDSAP384,
    keyPairAlgoECDSAP521,
    keyPairAlgoRSA2048,
    keyPairAlgoRSA4096,
    keyPairAlgoED25519,
];

interface license {
    ID:                 number,
    DatetimeCreated:    string,
    Active:             boolean,

    CreatedByUserID:     number,
    CreatedByAPIKeyID:   number,
    
    KeyPairID:          number, //used for identifying the keypair used to sign the license
    ContactName:        string,
    CompanyName:        string,
    PhoneNumber:        string,
    Email:              string,
    IssueDate:          string, //yyyy-mm-dd
    IssueTimestamp:     number, //unix timestamp in seconds
    ExpireDate:         string, //yyyy-mm-dd
    
    Signature:          string, //the encoded signature generated using the private key from the keypair, so we don't have to regernate it each time we want to redownload the license
    
    Verified:           boolean, //after a license is generated, we "read" it like a client app would and make sure it is valid before allowing it to be downloaded
    
    AppName:            string, //a copy of the value of this field at the time the license was created since it is part of the signature and we need it to redownload a license.
    FileFormat:         string, //yaml, json, etc.; copied from app when license is created
    ShowLicenseID:      boolean,
    ShowAppName:        boolean,

    //Calculated fields
    Expired:            boolean, //used when showing license data so we don't need to compare dates client side

    //JOINed fields
    KeyPairAlgoType:        string,
    CreatedByUsername:      string,
    CreatedByAPIKeyDescription: string,
    AppID:                  number,
    AppFileFormat:          string,
    AppDownloadFilename:    string,
    RenewedFromLicenseID:   number | null, //null when this license wasn't created by a renewal.
    RenewedToLicenseID:     number | null, //null when license hasn't been renewed.
}

interface user {
    ID:                 number,
    DatetimeCreated:    string,
    DatetimeModified:   string,
    CreatedByUserID:    number,
    Active:             boolean,

    Username:               string,
    Password:               string, //never used client side
    BadPasswordAttempts:    number, //never used client side

    Administrator:  boolean,
    CreateLicenses: boolean,
    ViewLicenses:   boolean,

    TwoFactorAuthEnabled:       boolean,
    TwoFactorAuthSecret:        string,
    TwoFactorAuthBadAttempts:   number,
}

interface downloadHistory {
    ID:                 number,
    DatetimeCreated:    string,
    TimestampCreated:   number,

    CreatedByUserID:    number,
    CreatedByAPIKeyID:  number,

    LicenseID:          number,

    //JOINed fields
    CreatedByUsername: string,
}

interface licenseNote {
    ID:                 number,
    DatetimeCreated:    string,
    TimestampCreated:   number,

    CreatedByUserID:    number,
    CreatedByAPIKeyID:  number,

    LicenseID:          number,
    Note:               string,

    //JOINed fields
    CreatedByUsername: string,
}
