/**
 * this.ts
 * 
 * This is used to manage a single license. This will display a license's
 * data, download history, and handle disabling or renewing a license.
 */

import { createApp } from "vue";
import { msgTypes, apiBaseURL, defaultTimeout, todayPlus, getToday, isEmail } from "./common";
import { get, post, handleRequestErrors, getJSON, handleAPIErrors } from "./fetch";
import { license, customFieldResults, downloadHistory, licenseNote, customFieldTypeInteger, customFieldTypeDecimal, customFieldTypeText, customFieldTypeBoolean, customFieldTypeMultiChoice, customFieldTypeDate } from "./types";

//This handles the main data and functionality for a license.
var manageLicense: any; //must be "any", not "ComponentPublicInstance" to remove errors when calling functions (methods) of this Vue instance.
if (document.getElementById("manageLicense")) {
    manageLicense = createApp({
        name: 'manageLicense',

        compilerOptions: {
            delimiters: ['[[', ']]'],
        },

        data() {
            return {
                //License data.
                licenseID: 0, //Populated in mounted with value from URL.
                licenseData: {} as license,
                licenseDataRetrieved: false, //true when data is retrieved from api, to stop alerts from showing until data is retrieved
                customFieldResults: [] as customFieldResults[],
                downloadHistory: [] as downloadHistory[],
                notes: [] as licenseNote[],

                //API request messages, for when API errors occur.
                msgLicenseData: '',
                msgLicenseDataType: '',
                msgHistory: '',
                msgHistoryType: '',
                msgNotes: '',
                msgNotesType: '',

                //Toggle to show more info for license.
                showAdvancedInfo: false,

                //Need these for v-if in html.
                customFieldTypeInteger: customFieldTypeInteger,
                customFieldTypeDecimal: customFieldTypeDecimal,
                customFieldTypeText: customFieldTypeText,
                customFieldTypeBoolean: customFieldTypeBoolean,
                customFieldTypeMultiChoice: customFieldTypeMultiChoice,
                customFieldTypeDate: customFieldTypeDate,

                //Endpoints.
                urls: {
                    getLicense: apiBaseURL + "licenses/",
                    getCustomFields: apiBaseURL + "custom-fields/results/",
                    getHistory: apiBaseURL + "licenses/history/",
                    getNotes: apiBaseURL + "licenses/notes/",

                    //Download license file link uses href, not url defined here.
                }
            }
        },
        methods: {
            //getLicense looks up the main data for a license.
            //
            //This ONLY looks up the base data for a license, not custom fields and
            //not download history or notes.
            //
            //We assume license ID is valid since it was validated when this page was 
            //loaded.
            getLicense: function () {
                let data: Object = {
                    id: this.licenseID,
                };
                fetch(get(this.urls.getLicense, data))
                    .then(handleRequestErrors)
                    .then(getJSON)
                    .then((j) => {
                        //Check if response is an error from the server.
                        let err: string = handleAPIErrors(j);
                        if (err !== "") {
                            this.msgLicenseData = err;
                            this.msgLicenseDataType = msgTypes.danger;
                            return;
                        }

                        this.licenseData = j.Data;
                        this.licenseDataRetrieved = true;

                        //pass license data to other Vue objects.
                        this.setLicenseDataInOtherVueObjects(j.Data);

                        return;
                    })
                    .catch((err) => {
                        console.log("fetch() error: >>", err, "<<");
                        this.msgLicenseData = "An unknown error occured. Please try again.";
                        this.msgLicenseDataType = msgTypes.danger;
                        return;
                    });

                return;
            },

            //getCustomFieldResults looks up the custom field results for a license.
            //
            //We assume license ID is valid since it was validated when this page was 
            //loaded.
            getCustomFieldResults: function () {
                let data: Object = {
                    licenseID: this.licenseID,
                };
                fetch(get(this.urls.getCustomFields, data))
                    .then(handleRequestErrors)
                    .then(getJSON)
                    .then((j) => {
                        //Check if response is an error from the server.
                        let err: string = handleAPIErrors(j);
                        if (err !== "") {
                            this.msgLicenseData = err;
                            this.msgLicenseDataType = msgTypes.danger;
                            return;
                        }

                        this.customFieldResults = j.Data || [];
                        return;
                    })
                    .catch((err) => {
                        console.log("fetch() error: >>", err, "<<");
                        this.msgLicenseData = "An unknown error occured. Please try again.";
                        this.msgLicenseDataType = msgTypes.danger;
                        return;
                    });

                return;
            },

            //getDownloadHistory looks up the download history for a license.
            //
            //We assume license ID is valid since it was validated when this page was 
            //loaded.
            getDownloadHistory: function () {
                let data: Object = {
                    licenseID: this.licenseID,
                };
                fetch(get(this.urls.getHistory, data))
                    .then(handleRequestErrors)
                    .then(getJSON)
                    .then((j) => {
                        //Check if response is an error from the server.
                        let err: string = handleAPIErrors(j);
                        if (err !== "") {
                            this.msgHistory = err;
                            this.msgHistoryType = msgTypes.danger;
                            return;
                        }

                        this.downloadHistory = j.Data || [];
                        return;
                    })
                    .catch((err) => {
                        console.log("fetch() error: >>", err, "<<");
                        this.msgHistory = 'An unknown error occured. Please try again.';
                        this.msgHistoryType = msgTypes.danger;
                        return;
                    });

                return;
            },

            //refreshDownloadHistory calls getDownloadHistory after a slight delay.
            //The delay is needed so that there is a high probability of the 
            //download occuring and being recorded.
            //
            //This function is called upon a user clicking one of the download buttons.
            refreshDownloadHistory: function () {
                const delay: number = 500;
                setTimeout(this.getDownloadHistory, delay);
                return;
            },

            //getNotes looks up the notes for this license.
            //
            //We assume license ID is valid since it was validated when this page was 
            //loaded.
            getNotes: function () {
                let data: Object = {
                    licenseID: this.licenseID,
                };
                fetch(get(this.urls.getNotes, data))
                    .then(handleRequestErrors)
                    .then(getJSON)
                    .then((j) => {
                        //Check if response is an error from the server.
                        let err: string = handleAPIErrors(j);
                        if (err !== "") {
                            this.msgNotes = err;
                            this.msgNotesType = msgTypes.danger;
                            return;
                        }

                        this.notes = j.Data || [];
                        return;
                    })
                    .catch((err) => {
                        console.log("fetch() error: >>", err, "<<");
                        this.msgNotes = 'An unknown error occured. Please try again.';
                        this.msgNotesType = msgTypes.danger;
                        return;
                    });

                return;
            },

            //passToNoteModal passes data about a chosen note to the modal so we can
            //display more info abou the note than we can in a table. This is done so
            //that we don't have to pass along just the note ID and then perform a
            //GET request to look up the note's data.
            // 
            //This is called when clicking the "view more" butotn for a note or
            //clicking the "add" button to add a new note.
            //
            //When adding, "undefined" is passed as the note.
            //
            //Note that this does not open the modal, that is handled through bootstrap
            //data-toggle and data-target attributes.
            passToNoteModal: function (n: licenseNote | undefined) {
                modalNote.setNoteInModal(n);
                return;
            },

            //setLicenseDataInOtherVueObjects sets the chosen license in Vue objects
            //that handle other parts of the GUI.
            //
            //This is called from getLicense().
            setLicenseDataInOtherVueObjects: function (lic: license) {
                modalDisableLicense.setLicenseID(lic.ID);

                modalNote.setLicenseID(lic.ID);

                modalRenewLicense.setLicenseData(lic.ID, lic.ExpirationDate);

                return;
            },
        },

        mounted() {
            //Get license ID from URL.
            let sp: URLSearchParams = new URLSearchParams(document.location.search);
            this.licenseID = parseInt(sp.get("id") as string);

            //Look up license data.
            this.getLicense();
            this.getCustomFieldResults();
            this.getDownloadHistory();
            this.getNotes();

            return;
        }
    }).mount("#manageLicense")
}

//This handles disabling a license. Disabled licenses cannot be further managed or
//used.
var modalDisableLicense: any; //must be "any", not "ComponentPublicInstance" to remove errors when calling functions (methods) of this Vue instance.
if (document.getElementById("modal-disableLicense")) {
    modalDisableLicense = createApp({
        name: 'modalDisableLicense',

        compilerOptions: {
            delimiters: ['[[', ']]'],
        },

        data() {
            return {
                //Set in setLicenseID().
                licenseID: 0,

                //Details about why license is being disabled, for future reference.
                note: "",

                //Form submission stuff.
                submitting: false,
                msgSave: "",
                msgSaveType: "",

                //Endpoint.
                urls: {
                    disableLicense: apiBaseURL + "licenses/disable/",
                },
            }
        },

        methods: {
            //setLicenseID sets the provided license ID in this Vue object.
            //
            //This is called from manageLicense.setLicenseDataInOtherVueObjects() upon
            //looking up data for the license.
            setLicenseID: function (licenseID: number) {
                this.licenseID = licenseID;
                return
            },

            //disableLicense makes the API call to mark a license as disabled/inactive.
            //This also saves a note about why the license is being disabled for more
            //historical information. After the API call completes successfully, the
            //Vue object that manages the base page is updated to mark the license as
            //disabled.
            disableLicense: function () {
                //Make sure we aren't already submitting.
                if (this.submitting) {
                    console.log("already submitting");
                    return;
                }

                //Validate.
                this.msgSaveType = msgTypes.danger;
                if (this.licenseID < 1) {
                    this.msgSave = "Could not determine which license you want to disabled.";
                    return;
                }
                if (this.note.trim() === "") {
                    this.msgSave = "You must provide a note describing why you are disabling this license.";
                    return;
                }

                //Make API requst.
                this.msgSaveType = msgTypes.primary;
                this.msgSave = "Disabling license...";
                this.submitting = true;

                //Perform api call.
                let data: Object = {
                    id: this.licenseID,
                    note: this.note,
                };
                fetch(post(this.urls.disableLicense, data))
                    .then(handleRequestErrors)
                    .then(getJSON)
                    .then((j) => {
                        //Check if response is an error from the server.
                        let err: string = handleAPIErrors(j);
                        if (err !== "") {
                            this.msgSave = err;
                            this.msgSaveType = msgTypes.danger;
                            this.submitting = false;
                            return;
                        }

                        //Call functions in other Vue objects to update page.
                        manageLicense.licenseData.Active = false;
                        manageLicense.getNotes();

                        //Show success message for modal. Don't unset the submitting
                        //field so user cannot resubmit again.
                        this.msgSave = "License disabled!";
                        this.msgSaveType = msgTypes.primary;

                        return;
                    })
                    .catch((err) => {
                        console.log("fetch() error: >>", err, "<<");
                        this.msgSave = "An unknown error occured. Please try again.";
                        this.msgSaveType = msgTypes.danger;
                        this.submitting = false;
                        return;
                    });

                return;
            },
        },
    }).mount("#modal-disableLicense");
}

//Add a note for a license, or view details of an existing note.
var modalNote: any; //must be "any", not "ComponentPublicInstance" to remove errors when calling functions (methods) of this Vue instance.
if (document.getElementById("modal-note")) {
    modalNote = createApp({
        name: 'modalNote',

        compilerOptions: {
            delimiters: ["[[", "]]"],
        },

        data() {
            return {
                //Set in setLicenseID().
                licenseID: 0,

                //The note's data.
                noteData: {
                    ID: 0,
                    LicenseID: 0,  //Set when data is submitted to server.
                    Note: "",      //Populated by user when adding or by initialize when viewing an existing note.
                } as licenseNote,

                //Form submission stuff.
                submitting: false,
                msgSave: "",
                msgSaveType: "",

                //Endpoint.
                urls: {
                    add: apiBaseURL + "licenses/notes/add/",
                },
            }
        },

        methods: {
            //setLicenseID sets the provided license ID in this Vue object.
            //
            //This is called from manageLicense.setLicenseDataInOtherVueObjects() upon
            //looking up data for the license.
            setLicenseID: function (licenseID: number) {
                this.licenseID = licenseID;
                return
            },

            //setNoteInModal saves the provided note in this Vue object.
            //
            //This is called when a user clicks button to view details of a note or
            //when user clicks button to add a note. See manageLicense.passToNoteModal().
            setNoteInModal: function (n: licenseNote | undefined) {
                //Always reset modal.
                this.resetForm();

                //Handle user wanting to add a note.
                if (n === undefined) {
                    return;
                }

                //Handle user wanting to view details of a note.
                this.noteData = n;
                return;
            },

            //resetForm sets the modal to a blank state. This function is called
            //after a note is saved and when the modal is launched.
            resetForm: function () {
                this.noteData = {
                    ID: 0,
                    LicenseID: 0,
                    Note: "",
                    CreatedByUsername: "",
                    DatetimeCreated: "",
                } as licenseNote;

                this.msgSave = "";
                this.msgSaveType = "";

                return;
            },

            //add saves a note and updates the list of notes displayed.
            add: function () {
                //Make sure we aren't already submitting.
                if (this.submitting) {
                    console.log("already submitting");
                    return;
                }

                //Validate.
                this.msgSaveType = msgTypes.danger;
                if (this.licenseID < 1) {
                    this.msgSave = "Could not determine which license you want to add a note for.";
                    return;
                }
                if (this.noteData.Note.trim() === "") {
                    this.msgSave = "You must provide a note.";
                    return;
                }

                //Make API request.
                this.msgSaveType = msgTypes.primary;
                this.msgSave = "Adding note...";
                this.submitting = true;

                this.noteData.LicenseID = this.licenseID;

                let data: Object = {
                    data: JSON.stringify(this.noteData),
                };
                fetch(post(this.urls.add, data))
                    .then(handleRequestErrors)
                    .then(getJSON)
                    .then((j) => {
                        //Check if response is an error from the server.
                        let err: string = handleAPIErrors(j);
                        if (err !== "") {
                            this.msgSave = err;
                            this.msgSaveType = msgTypes.danger;
                            this.submitting = false;
                            return;
                        }

                        //Update page to show new note.
                        manageLicense.getNotes();

                        //Show success message for modal.
                        this.msgSave = "Note added!";
                        this.msgSaveType = msgTypes.primary;
                        setTimeout(() => {
                            this.msgSave = "";
                            this.msgSaveType = "";
                            this.submitting = false;
                            this.resetForm();
                        }, defaultTimeout)

                        return;
                    })
                    .catch((err) => {
                        console.log("fetch() error: >>", err, "<<");
                        this.msgSave = 'An unknown error occured. Please try again.';
                        this.msgSaveType = msgTypes.danger;
                        this.submitting = false;
                        return;
                    });

                return;
            },
        },
    }).mount("#modal-note")
}

//Renew a license. This marks the current license as no longer active and creates a
//new license to replace it.
var modalRenewLicense: any; //must be "any", not "ComponentPublicInstance" to remove errors when calling functions (methods) of this Vue instance.
if (document.getElementById("modal-renewLicense")) {
    modalRenewLicense = createApp({
        name: 'modalRenewLicense',

        compilerOptions: {
            delimiters: ["[[", "]]"],
        },

        data() {
            return {
                //Set in setLicenseID().
                licenseID: 0,
                currentExpirationDate: "",

                //Data for new license.
                newExpirationDate: "",

                //Set to true upon successful renewal api call.
                renewed: false,

                //Form submission stuff.
                submitting: false,
                msgSave: "",
                msgSaveType: "",

                //Endpoint.
                urls: {
                    renew: apiBaseURL + "licenses/renew/",
                },
            }
        },

        computed: {
            //minDate creates the min value for the date picker for the new expiration
            //date. We want the min value a user can pick to be after the current
            //expiration date.
            minDate: function () {
                let ymd = this.currentExpirationDate;

                let split: string[] = ymd.split("-");
                let yIn: number = parseInt(split[0]);
                let mIn: number = parseInt(split[1]) - 1; //adjust month to zero based.
                let dIn: number = parseInt(split[2]);

                let input: Date = new Date(yIn, mIn, dIn);
                let added: Date = new Date(input.setDate(input.getDate() + 1));

                let y: number = added.getFullYear();
                let m: number = added.getMonth() + 1;
                let d: number = added.getDate();
                let yy: string = y.toString();
                let mm: string = (m < 10) ? "0" + m.toString() : m.toString();
                let dd: string = (d < 10) ? "0" + d.toString() : d.toString();

                return yy + "-" + mm + "-" + dd;
            },
        },

        methods: {
            //setLicenseData sets the provided license ID in this Vue object. This 
            //also saves the license's current expiration date for displaying and
            //comparing against a new expiration date.
            //
            //This is called from manageLicense.setLicenseDataInOtherVueObjects() upon
            //looking up data for the license.
            setLicenseData: function (licenseID: number, currentExpirationDate: string) {
                this.licenseID = licenseID;
                this.currentExpirationDate = currentExpirationDate;
                return
            },

            //renew handles renewing a license. This license's data is copied to
            //a new license with a new expiration date.
            renew: function () {
                //Make sure we aren't already submitting.
                if (this.submitting) {
                    console.log("already submitting");
                    return;
                }

                //Validate.
                this.msgSaveType = msgTypes.danger;
                if (this.licenseID < 1) {
                    this.msgSave = "Could not determine which license you want to renew.";
                    return;
                }
                if (this.newExpirationDate === "") {
                    this.msgSave = "You must provide the new expiration date.";
                    return;
                }

                //Make API request.
                this.msgSaveType = msgTypes.primary;
                this.msgSave = "Renewing license...";
                this.submitting = true;

                let data: Object = {
                    id: this.licenseID,
                    newExpirationDate: this.newExpirationDate,
                };
                fetch(post(this.urls.renew, data))
                    .then(handleRequestErrors)
                    .then(getJSON)
                    .then((j) => {
                        //Check if response is an error from the server.
                        let err: string = handleAPIErrors(j);
                        if (err !== "") {
                            this.msgSave = err;
                            this.msgSaveType = msgTypes.danger;
                            this.submitting = false;
                            return;
                        }

                        //Show success message for modal.
                        this.msgSave = "License Renewed!";
                        this.renewed = true;

                        //Redirect user to renewed license's page.
                        let licenseID: number = j.Data;
                        if (licenseID === undefined) {
                            this.msgSave = "Could not determine where to redirect you. This is an odd error...";
                            this.msgSaveType = msgTypes.warning;
                            return;
                        }
                        setTimeout(() => {
                            window.location.href = "/app/licensing/license/?id=" + licenseID;
                        }, defaultTimeout);

                        return;
                    })
                    .catch((err) => {
                        console.log("fetch() error: >>", err, "<<");
                        this.msgSave = "An unknown error occured. Please try again.";
                        this.msgSaveType = msgTypes.danger;
                        this.submitting = false;
                        return;
                    });

                return;
            },
        },
    }).mount("#modal-renewLicense");
}