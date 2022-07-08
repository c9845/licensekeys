/**
 * tools.ts
 * The code in this is used on the administrative tools page. These tools are used to
 * perform low-level app or database maintenance.
 */

/// <reference path="common.ts" />
/// <reference path="fetch.ts" />
/// <reference path="types.ts" />

if (document.getElementById("toolsClearActivityLog")) {
    //toolsClearActivityLog is used to clear the activity log of old data. This can be
    //helpful for cleaning up very old data or when the activity log gets very big.
    //@ts-ignore cannot find name Vue
    var toolsClearActivityLog = new Vue({
        name: 'toolsClearActivityLog',
        delimiters: ['[[', ']]'],
        el: '#toolsClearActivityLog',
        data: {
            priorToDate:    '', //yyyy-mm-dd
            msg:            '',
            msgType:        '',
            submitting:     false,

            //Endpoints.
            urls: {
                clear: "/api/activity-log/clear/",
            }
        },
        methods: {
            clear: function() {
                //Validate.
                if (this.priorToDate === "") {
                    this.msg = "Invalid date provided.";
                    this.msgType = msgTypes.danger;
                    return;
                }
    
                //Validation ok.
                this.msg = 'Working...  This can take a while.';
                this.msgType = msgTypes.primary;
                this.submitting = true;
    
                //Perform api call.
                let data: Object = {
                    priorToDate: this.priorToDate
                };
                fetch(post(this.urls.clear, data))
                .then(handleRequestErrors)
                .then(getJSON)
                .then(function (j) {
                    //check if response is an error from the server
                    let err: string = handleAPIErrors(j);
                    if (err !== '') {
                        toolsClearActivityLog.msg =         err;
                        toolsClearActivityLog.msgType =     msgTypes.danger;
                        toolsClearActivityLog.submitting =  false;
                        return;
                    }
    
                    toolsClearActivityLog.msg =     "Done! Rows deleted: " + j.Data;
                    toolsClearActivityLog.msgType = msgTypes.success;
                    setTimeout(function () {
                        toolsClearActivityLog.msg =     '';
                        toolsClearActivityLog.msgType = '';
                    }, defaultTimeout * 3);
    
                    toolsClearActivityLog.submitting = false;
                    return;
                })
                .catch(function (err) {
                    console.log("fetch() error: >>", err, "<<");
                    toolsClearActivityLog.msg =         'An unknown error occured.  Please try again.';
                    toolsClearActivityLog.msgType =     msgTypes.danger;
                    toolsClearActivityLog.submitting =  false;
                    return;
                });
    
                return;
            },
        },
    });
}
