/**
 * header.ts
 * This deals with toggling the username to/from the Logout text.
*/

/// <reference path="common.ts" />

if (document.getElementById("headerBtns")) {
    //@ts-ignore cannot find name Vue
    var headerBtns = new Vue({
        name: 'headerBtns',
        delimiters: ['[[', ']]'],
        el: '#headerBtns',
        data: {
            usernameHovered: false, //usernameHovered is used to change the text from username to Logout
        },
    });
}
