var XMLHttpRequest = require("xmlhttprequest").XMLHttpRequest;
const req = new XMLHttpRequest();
req.onreadystatechange = function() {
    if (req.readyState == XMLHttpRequest.DONE) {
        console.log(req.response)
        //let articles = JSON.parse() <-- get result from POST request and save as JSON
        jswin = window.open("", "jswin", "width=550,height=450");
        /*for (article of articles) {
            jswin.document.write(article);
        }*/
        
    }
}
const params = { "num": 10 };
const url = "https://tartanhackathon.uc.r.appspot.com/recommend?num=10";
var result = "hello"
var query = {
    "texts": [
        {
            "body": `My point is that writing a new operating system that is closely tied to any
particular piece of hardware, especially a weird one like the Intel line,
is basically wrong.  An OS itself should be easily portable to new hardware
platforms.  When OS/360 was written in assembler for the IBM 360
25 years ago, they probably could be excused.  When MS-DOS was written
specifically for the 8088 ten years ago, this was less than brilliant, as
IBM and Microsoft now only too painfully realize. Writing a new OS only for the
386 in 1991 gets you your second 'F' for this term.  But if you do real well
on the final exam, you can still pass the course.`
        }
    ]
};
req.open('POST', url, false);
req.setRequestHeader('Content-Type', "application/json")
var data = JSON.stringify(query);
console.log(data)
req.send(data)
console.log(req)
var articles = JSON.parse(req.responseText)
//jswin = window.open("", "jswin", "width=550,height=450");
for (article of articles["urls"]) {
    //jswin.document.write(article + "\n");
    console.log(article)
}