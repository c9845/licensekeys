/**
 * manageLicense.ts
 * This is used to manage a single license. This will display a license's
 * data, download history, and handle disabling or renewing a license.
 */

/// <reference path="common.ts" />
/// <reference path="fetch.ts" />
/// <reference path="types.ts" />

if (document.getElementById("manageLicense")) {
    //@ts-ignore cannot find name Vue
    var manageLicense = new Vue({
        name: 'manageLicense',
        delimiters: ['[[', ']]'],
        el: '#manageLicense',
        data: {
            licenseID: 0, //populated in mounted with value from input
            licenseData: {} as license,
            licenseDataRetrieved: false, //true when data is retrieved from api, to stop alerts from showing until data is retrieved
            customFieldResults: [] as customFieldResults[],
            downloadHistory: [] as downloadHistory[],
            notes: [] as licenseNote[],

            msgLicenseData: '',
            msgLicenseDataType: '',
            msgHistory: '',
            msgHistoryType: '',
            msgNotes: '',
            msgNotesType: '',

            showAdvancedInfo: false, //set by button click

            //need these for v-if in html
            customFieldTypeInteger: customFieldTypeInteger,
            customFieldTypeDecimal: customFieldTypeDecimal,
            customFieldTypeText: customFieldTypeText,
            customFieldTypeBoolean: customFieldTypeBoolean,
            customFieldTypeMultiChoice: customFieldTypeMultiChoice,
            customFieldTypeDate: customFieldTypeDate,

            //endpoints
            urls: {
                getLicense: "/api/licenses/",
                getCustomFields: "/api/custom-fields/results/",
                getHistory: "/api/licenses/history/",
                getNotes: "/api/licenses/notes/",
                //download license file used href, not url defined here.
            }
        },
        methods: {
            //getLicense looks up the common data for a license.
            //We assume license ID is valid since it was validated when
            //this page was loaded.
            getLicense: function () {
                let data: Object = {
                    id: this.licenseID,
                };
                fetch(get(this.urls.getLicense, data))
                    .then(handleRequestErrors)
                    .then(getJSON)
                    .then(function (j) {
                        //check if response is an error from the server
                        let err: string = handleAPIErrors(j);
                        if (err !== '') {
                            manageLicense.msgLicenseData = err;
                            manageLicense.msgLicenseDataType = msgTypes.danger;
                            return;
                        }

                        manageLicense.licenseData = j.Data;
                        manageLicense.licenseDataRetrieved = true;

                        //pass license data to other Vue objects.
                        manageLicense.passData();

                        return;
                    })
                    .catch(function (err) {
                        console.log("fetch() error: >>", err, "<<");
                        manageLicense.msgLicenseData = 'An unknown error occured.  Please try again.';
                        manageLicense.msgLicenseDataType = msgTypes.danger;
                        return;
                    });

                return;
            },

            //getCustomFieldResults looks up the custom field results for a license.
            //We assume license ID is valid since it was validated when
            //this page was loaded.
            getCustomFieldResults: function () {
                let data: Object = {
                    licenseID: this.licenseID,
                };
                fetch(get(this.urls.getCustomFields, data))
                    .then(handleRequestErrors)
                    .then(getJSON)
                    .then(function (j) {
                        //check if response is an error from the server
                        let err: string = handleAPIErrors(j);
                        if (err !== '') {
                            manageLicense.msgLicenseData = err;
                            manageLicense.msgLicenseDataType = msgTypes.danger;
                            return;
                        }

                        manageLicense.customFieldResults = j.Data || [];
                        return;
                    })
                    .catch(function (err) {
                        console.log("fetch() error: >>", err, "<<");
                        manageLicense.msgLicenseData = 'An unknown error occured.  Please try again.';
                        manageLicense.msgLicenseDataType = msgTypes.danger;
                        return;
                    });

                return;
            },

            //getDownloadHistory looks up the download history for a license.
            //We assume license ID is valid since it was validated when
            //this page was loaded.
            getDownloadHistory: function () {
                let data: Object = {
                    licenseID: this.licenseID,
                };
                fetch(get(this.urls.getHistory, data))
                    .then(handleRequestErrors)
                    .then(getJSON)
                    .then(function (j) {
                        //check if response is an error from the server
                        let err: string = handleAPIErrors(j);
                        if (err !== '') {
                            manageLicense.msgHistory = err;
                            manageLicense.msgHistoryType = msgTypes.danger;
                            return;
                        }

                        manageLicense.downloadHistory = j.Data || [];
                        return;
                    })
                    .catch(function (err) {
                        console.log("fetch() error: >>", err, "<<");
                        manageLicense.msgHistory = 'An unknown error occured.  Please try again.';
                        manageLicense.msgHistoryType = msgTypes.danger;
                        return;
                    });

                return;
            },

            //refreshDownloadHistory calls getDownloadHistory after a slight delay.
            //The delay is needed so that there is a high probability of the 
            //download occuring and being recorded. This function is called upon a
            //user clicking one of the download buttons.
            refreshDownloadHistory: function () {
                const delay: number = 500;
                setTimeout(manageLicense.getDownloadHistory, delay);
                return;
            },

            //getNotes looks up the notes for this license.
            //We assume license ID is valid since it was validated when
            //this page was loaded.
            getNotes: function () {
                let data: Object = {
                    licenseID: this.licenseID,
                };
                fetch(get(this.urls.getNotes, data))
                    .then(handleRequestErrors)
                    .then(getJSON)
                    .then(function (j) {
                        //check if response is an error from the server
                        let err: string = handleAPIErrors(j);
                        if (err !== '') {
                            manageLicense.msgNotes = err;
                            manageLicense.msgNotesType = msgTypes.danger;
                            return;
                        }

                        manageLicense.notes = j.Data || [];
                        return;
                    })
                    .catch(function (err) {
                        console.log("fetch() error: >>", err, "<<");
                        manageLicense.msgNotes = 'An unknown error occured.  Please try again.';
                        manageLicense.msgNotesType = msgTypes.danger;
                        return;
                    });

                return;
            },

            //setNoteModal is called when a user clicks the button to open the note
            //modal, either for adding a new note or viewing details of an existing
            //note. When clicking the add button, the input is undefined. When
            //clicked to view an existing note, the input is the note's data for
            //populating the modal with.
            setNoteModal: function (n: licenseNote | undefined) {
                modalNote.initialize(n);
                return;
            },

            //passData passes license data to other Vue objects. Other Vue objects
            //may need certain license data and this function centralizes all the
            //data passed from thie Vue object to other Vue objects. This is called
            //after a license's data is retrieved.
            passData: function () {
                modalDisableLicense.licenseID = this.licenseID;

                modalNote.licenseID = this.licenseID;

                modalRenewLicense.licenseID = this.licenseID;
                modalRenewLicense.currentExpireDate = this.licenseData.ExpireDate;

                return;
            },
        },
        mounted() {
            //get license ID from input.
            let id: number = parseInt((document.getElementById("licenseID") as HTMLInputElement).value);
            this.licenseID = id;

            //look up license data
            this.getLicense();
            this.getCustomFieldResults();
            this.getDownloadHistory();
            this.getNotes();

            return;
        }
    });
}

if (document.getElementById("modal-disableLicense")) {
    //@ts-ignore cannot find name Vue
    var modalDisableLicense = new Vue({
        name: 'modalDisableLicense',
        delimiters: ['[[', ']]'],
        el: '#modal-disableLicense',
        data: {
            licenseID: 0,  //set in manageLicense.passData().
            note: "", //details about why license is being disabled

            submitting: false,
            msgSave: "",
            msgSaveType: "",

            //endpoint
            urls: {
                disableLicense: "/api/licenses/disable/",
            },
        },
        methods: {
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

                //Validate
                this.msgSaveType = msgTypes.danger;
                if (this.licenseID < 1) {
                    this.msgSave = "Could not determine which license you want to disabled.";
                    return;
                }
                if (this.note.trim() === "") {
                    this.msgSave = "You must provide a note describing why you are disabling this license.";
                    return;
                }

                //Validation ok.
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
                    .then(function (j) {
                        //check if response is an error from the server
                        let err: string = handleAPIErrors(j);
                        if (err !== '') {
                            modalDisableLicense.msgSave = err;
                            modalDisableLicense.msgSaveType = msgTypes.danger;
                            modalDisableLicense.submitting = false;
                            return;
                        }

                        //Update page to show license as disabled.
                        manageLicense.licenseData.Active = false;
                        manageLicense.getNotes();

                        //Show success message for modal. Don't unset the submitting
                        //field so user cannot resubmit again.
                        modalDisableLicense.msgSave = "License disabled!";
                        modalDisableLicense.msgSaveType = msgTypes.primary;

                        return;
                    })
                    .catch(function (err) {
                        console.log("fetch() error: >>", err, "<<");
                        modalDisableLicense.msgSave = 'An unknown error occured. Please try again.';
                        modalDisableLicense.msgSaveType = msgTypes.danger;
                        modalDisableLicense.submitting = false;
                        return;
                    });

                return;
            },
        },
    });
}

if (document.getElementById("modal-note")) {
    //@ts-ignore cannot find name Vue
    var modalNote = new Vue({
        name: 'modalNote',
        delimiters: ['[[', ']]'],
        el: '#modal-note',
        data: {
            licenseID: 0, //set in manageLicense.passData().

            noteData: {
                LicenseID: 0,  //set when data is submitted to server from licenseID.
                Note: "", //populated by user when adding or by initialize when viewing an existing note.
            } as licenseNote,

            addingNew: false, //set in initialize based on if user is viewing an exiting note or adding a new note.

            submitting: false,
            msgSave: "",
            msgSaveType: "",

            //endpoint
            urls: {
                add: "/api/licenses/notes/add/",
            },
        },
        computed: {
            //modalTitle sets the title of the modal.
            modalTitle: function () {
                if (this.addingNew) {
                    return "Add Note";
                }

                return "Note Details";
            },
        },
        methods: {
            //initialize is called when a user clicks on a button to open the modal
            //by the manageLicenses.setNoteModal function. When adding a new note
            //this resets the modal to a clean state. When viewing an existing modal
            //this populates the Vue object with the chosen note's data.
            initialize: function (n: licenseNote | undefined) {
                //Always reset, even when viewing an existing note, so that we can
                //be sure no old data is present.
                this.resetModal();

                //User wants to add a note.
                if (n === undefined) {
                    this.addingNew = true;
                    return;
                }

                //User wants to view an existing note.
                this.noteData = n;
                this.addingNew = false;

                return;
            },

            //resetModal sets a modal to a blank state. This function is called
            //after a note is saved and when the modal is launched.
            resetModal: function () {
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

                //Validate
                this.msgSaveType = msgTypes.danger;
                if (this.licenseID < 1) {
                    this.msgSave = "Could not determine which license you want to add a note for.";
                    return;
                }
                if (this.noteData.Note.trim() === "") {
                    this.msgSave = "You must provide a note.";
                    return;
                }

                //Validation ok.
                this.msgSaveType = msgTypes.primary;
                this.msgSave = "Adding note...";
                this.submitting = true;

                //Set licenseID into note object.
                this.noteData.LicenseID = this.licenseID;

                //Perform api call.
                let data: Object = {
                    data: JSON.stringify(this.noteData),
                };
                fetch(post(this.urls.add, data))
                    .then(handleRequestErrors)
                    .then(getJSON)
                    .then(function (j) {
                        //check if response is an error from the server
                        let err: string = handleAPIErrors(j);
                        if (err !== '') {
                            modalNote.msgSave = err;
                            modalNote.msgSaveType = msgTypes.danger;
                            modalNote.submitting = false;
                            return;
                        }

                        //Update page to show new note.
                        manageLicense.getNotes();

                        //Show success message for modal.
                        modalNote.msgSave = "Note added!";
                        modalNote.msgSaveType = msgTypes.primary;
                        setTimeout(function () {
                            modalNote.msgSave = "";
                            modalNote.msgSaveType = "";
                            modalNote.submitting = false;
                            modalNote.resetModal();
                        }, defaultTimeout)

                        return;
                    })
                    .catch(function (err) {
                        console.log("fetch() error: >>", err, "<<");
                        modalNote.msgSave = 'An unknown error occured. Please try again.';
                        modalNote.msgSaveType = msgTypes.danger;
                        modalNote.submitting = false;
                        return;
                    });

                return;
            },
        },
    });
}

if (document.getElementById("modal-renewLicense")) {
    //@ts-ignore cannot find name Vue
    var modalRenewLicense = new Vue({
        name: 'modalRenewLicense',
        delimiters: ['[[', ']]'],
        el: '#modal-renewLicense',
        data: {
            licenseID: 0, //set in manageLicense.passData().
            currentExpireDate: "", //set in manageLicense.passData().
            newExpireDate: "",
            renewed: false, //set to true upon successful renewal api call.

            submitting: false,
            msgSave: "",
            msgSaveType: "",

            //endpoint
            urls: {
                renew: "/api/licenses/renew/",
            },
        },
        computed: {
            //minDate creates the min value for the date picker for the new expiration
            //date. We want the min value a user can pick to be after the current
            //expiration date.
            minDate: function () {
                return dateAdd(this.currentExpireDate, 1);
            },
        },
        methods: {
            //renew handles renewing a license. This license's data is copied to
            //a new license with a new expiration date.
            renew: function () {
                //Make sure we aren't already submitting.
                if (this.submitting) {
                    console.log("already submitting");
                    return;
                }

                //Validate
                this.msgSaveType = msgTypes.danger;
                if (this.licenseID < 1) {
                    this.msgSave = "Could not determine which license you want to renew.";
                    return;
                }
                if (this.newExpireDate === "") {
                    this.msgSave = "You must provide the new expiration date.";
                    return;
                }

                //Validation ok.
                this.msgSaveType = msgTypes.primary;
                this.msgSave = "Renewing license...";
                this.submitting = true;

                //Perform api call.
                let data: Object = {
                    id: this.licenseID,
                    newExpireDate: this.newExpireDate,
                };
                fetch(post(this.urls.renew, data))
                    .then(handleRequestErrors)
                    .then(getJSON)
                    .then(function (j) {
                        //check if response is an error from the server
                        let err: string = handleAPIErrors(j);
                        if (err !== '') {
                            modalRenewLicense.msgSave = err;
                            modalRenewLicense.msgSaveType = msgTypes.danger;
                            modalRenewLicense.submitting = false;
                            return;
                        }

                        //Show success message for modal.
                        modalRenewLicense.msgSave = "License Renewed!";
                        modalRenewLicense.renewed = true;

                        //Redirect user to renewed license's page.
                        let licenseID: number = j.Data;
                        if (licenseID === undefined) {
                            createLicense.msg = "Could not determine where to redirect you. This is an odd error...";
                            createLicense.msgType = msgTypes.warning;
                            return;
                        }
                        setTimeout(function () {
                            window.location.href = "/app/licensing/license/?id=" + licenseID;
                        }, defaultTimeout);

                        return;
                    })
                    .catch(function (err) {
                        console.log("fetch() error: >>", err, "<<");
                        modalRenewLicense.msgSave = 'An unknown error occured. Please try again.';
                        modalRenewLicense.msgSaveType = msgTypes.danger;
                        modalRenewLicense.submitting = false;
                        return;
                    });

                return;
            },
        },
        mounted() {
            //get license ID from input.
            let id: number = parseInt((document.getElementById("licenseID") as HTMLInputElement).value);
            this.licenseID = id;

            return;
        }
    });
}