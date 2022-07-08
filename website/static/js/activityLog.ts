/**
 * activityLog.ts
 * This simply is used to show or hide the help message for the activity log.
 */

/// <reference path="common.ts" />

if (document.getElementById("activityLog")) {
    //activityLog is just used to toggle showing the help text.
    //@ts-ignore cannot find name Vue
    var activityLog = new Vue({
        name: 'activityLog',
        delimiters: ['[[', ']]'],
        el: '#activityLog',
        data: {
            showHelp: false,
        },
    });
}
