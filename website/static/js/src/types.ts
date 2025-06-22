/**
 * types.ts
 * 
 * The stuff in this file should generally match the structs defined for each 
 * database table.
 * 
 * Order of export interfaces matches order of db-*.go files.
 */

//activityLog handles interacting with the log of user activity, for auditing purposes.
export interface activityLog {
    ID: number,
    DatetimeCreated: string,
    TimestampCreated: number, //unix timestamp in nanoseconds
    Method: string,
    URL: string,
    RemoteIP: string,
    UserAgent: string,
    TimeDuration: number,
    PostFormValues: string,

    UserID: number, //possibly null
    APIKeyID: number, //possibly null

    //JOINed fields
    Username: string,
    APIKeyDescription: string,
    APIKeyK: string,

    //Calculated fields.
    DatetimeCreatedInTZ: string,
}

//apiKey handles interacting with API keys.
export interface apiKey {
    ID: number,
    DatetimeCreated: string,
    DatetimeModified: string,
    Active: boolean,
    CreatedByUserID: number,

    Description: string,
    K: string,

    CreateLicense: boolean,
    RevokeLicense: boolean,
    DownloadLicense: boolean,

    //JOINed fields
    CreatedByUsername: string,

    //Calculated fields.
    DatetimeCreatedInTZ: string,
}

//app is an app you create licenses for.
export interface app {
    ID: number,
    DatetimeCreated: string,
    DatetimeModified: string,
    CreatedByUserID: number,
    Active: boolean,

    Name: string,
    DaysToExpiration: number,  //the number of days to add on to "today" to calculate a default expiration date of a license

    ShowLicenseID: boolean, //if the ID field of a created license file will be populated/non-zero.
    ShowAppName: boolean, //if the Application field of a created license file will be populated/non-blank.

    FileFormat: string,  //json.

    DownloadFilename: string,
}

//appSettings handles interacting with the settings that change functionality of the
//app.
export interface appSettings {
    ID: number,
    DatetimeModified: string,

    EnableActivityLog: boolean,
    AllowAPIAccess: boolean,

    Allow2FactorAuth: boolean,
    Force2FactorAuth: boolean,
    ForceSingleSession: boolean,
}

//custom fields are extra data added to a license. This stores the definition of each
//field.
export interface customFieldDefined {
    ID: number,
    DatetimeCreated: string,
    DatetimeModified: string,
    CreatedByUserID: number,
    Active: boolean,

    AppID: number,
    Type: string,
    Name: string,
    Instructions: string, //what field is for and expected value

    IntegerDefaultValue: number,
    DecimalDefaultValue: number,
    TextDefaultValue: string,
    BoolDefaultValue: boolean,
    MultiChoiceDefaultValue: string,
    DateDefaultIncrement: number, //number of days incremented from "today", date license is being created

    NumberMinValue: number,
    NumberMaxValue: number,
    MultiChoiceOptions: string,

    //When saving a license, we retrieve the defined fields for an app 
    //and set the value for each field using the same list of objects 
    //returned just for ease of use and not changing types. Therefore,
    //we need fields in this type/object to store the chosen/provided
    //values for each field. Only one of these fields is populated per
    //the Type field.
    IntegerValue: number,
    DecimalValue: number,
    TextValue: string,
    BoolValue: boolean,
    MultiChoiceValue: string,
    DateValue: string,
}

export const customFieldTypeInteger: string = "Integer";
export const customFieldTypeDecimal: string = "Decimal";
export const customFieldTypeText: string = "Text";
export const customFieldTypeBoolean: string = "Boolean";
export const customFieldTypeMultiChoice: string = "Multi-Choice";
export const customFieldTypeDate: string = "Date";
export const customFieldTypes: string[] = [
    customFieldTypeInteger,
    customFieldTypeDecimal,
    customFieldTypeText,
    customFieldTypeMultiChoice,
    customFieldTypeBoolean,
    customFieldTypeDate,
];

//this stores the data for each custom field for each license that has been created.
export interface customFieldResults {
    ID: number,
    DatetimeCreated: string,
    Active: boolean,

    CreatedByUserID: number,
    CreatedByAPIKeyID: number,

    LicenseID: number, //what license this result is for
    CustomFieldDefinedID: number, //just for reference, although we will probably never need to refer back to defined field once license is created.
    CustomFieldType: string, //integer, decimal, text, etc.
    CustomFieldName: string, //so that if defined field's name changes, we still have name as was set when license was created.

    //only one of these has a valid value, per CustomFieldType
    IntegerValue: number,
    DecimalValue: number,
    TextValue: string,
    BoolValue: boolean,
    MultiChoiceValue: string,
    DateValue: string,
}

//key pairs the public-private key pairs used to sign and verifiy a license.
export interface keyPair {
    ID: number,
    DatetimeCreated: string,
    DatetimeModified: string,
    CreatedByUserID: number,
    Active: boolean,

    AppID: number,
    Name: string,
    PrivateKey: string, //should NEVER be returned
    PublicKey: string, //returned so user can copy-paste into their app's code
    PrivateKeyEncrypted: boolean, //whether the private key is stored in plaintext or encrypted

    KeypairAlgo: string, //ed25519.
    FingerprintAlgo: string, //sha512.
    EncodingAlgo: string, //base64

    IsDefault: boolean, //if this is the default keypair for the app
}

//license is a license created for an app.
export interface license {
    ID: number,
    DatetimeCreated: string,
    Active: boolean,

    CreatedByUserID: number,
    CreatedByAPIKeyID: number,

    AppID: number,
    KeyPairID: number,
    ContactName: string,
    CompanyName: string,
    PhoneNumber: string,
    Email: string,
    IssueDate: string, //yyyy-mm-dd
    IssueTimestamp: number, //unix timestamp in seconds
    ExpirationDate: string, //yyyy-mm-dd

    Signature: string, //the encoded signature generated using the private key from the keypair, so we don't have to regernate it each time we want to redownload the license

    Verified: boolean, //after a license is generated, we "read" it like a client app would and make sure it is valid before allowing it to be downloaded

    AppName: string, //a copy of the value of this field at the time the license was created since it is part of the signature and we need it to redownload a license.
    ShowLicenseID: boolean,
    ShowAppName: boolean,
    FileFormat: string, //json

    //Calculated fields
    Expired: boolean, //used when showing license data so we don't need to compare dates client side
    DatetimeCreatedInTZ: string,
    IssueDateInTZ: string,
    Timezone: string,

    //JOINed fields
    KeyPairAlgo: string,
    FingerprintAlgo: string,
    EncodingAlgo: string,
    CreatedByUsername: string,
    CreatedByAPIKeyDescription: string,
    AppDownloadFilename: string,
    RenewedFromLicenseID: number | null, //null when this license wasn't created by a renewal.
    RenewedToLicenseID: number | null, //null when license hasn't been renewed.
}

//user handles interacting with authenticated users of the app.
export interface user {
    ID: number,
    DatetimeCreated: string,
    DatetimeModified: string,
    Active: boolean,
    CreatedByUserID: number,

    Username: string, //email address.
    Password: string, //SHOULD ALWAYS BE BLANK! DON'T USE THIS FIELD!
    BadPasswordAttempts: number, //never used client side
    Fname: string,
    Lname: string,

    Administrator: boolean,
    CreateLicenses: boolean,
    ViewLicenses: boolean,

    TwoFactorAuthEnabled: boolean,
    TwoFactorAuthSecret: string, //SHOULD ALWAYS BE BLANK! DON'T USE THIS FIELD!
    TwoFactorAuthBadAttempts: number, //SHOULD ALWAYS BE BLANK! DON'T USE THIS FIELD!

    //Used when setting password.
    PasswordInput1: string,
    PasswordInput2: string,
}

//store the download history of each license for diagnostics and auditing.
export interface downloadHistory {
    ID: number,
    DatetimeCreated: string,
    TimestampCreated: number,

    CreatedByUserID: number,
    CreatedByAPIKeyID: number,

    LicenseID: number,

    //JOINed fields
    CreatedByUsername: string,
}

//notes store extra data about a license for future reference.
export interface licenseNote {
    ID: number,
    DatetimeCreated: string,
    TimestampCreated: number,

    CreatedByUserID: number,
    CreatedByAPIKeyID: number,

    LicenseID: number,
    Note: string,

    //JOINed fields
    CreatedByUsername: string,
}
