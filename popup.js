const req = new XMLHttpRequest();
const params = { "num": 10 };
const url = "https://tartanhackathon.uc.r.appspot.com/recommend?num=10";
function getText() {
    return document.body.innerText
}
localStorage.setItem('counter', 0);
document.addEventListener('DOMContentLoaded', function () {
    var checkPageButton = document.getElementById('checkPage');
    checkPageButton.addEventListener('click', function (e) {
        e.preventDefault();

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
        localStorage.setItem('counter', parseInt(localStorage.getItem('counter')) + 1);
        chrome.tabs.executeScript(null, {
            code: `document.all[0].innerText`,
            allFrames: false, // this is the default
            runAt: 'document_start',
        }, function (results) {
            var result = results[0];
            var query = {
                "texts": [
                    {
                        "body": result
                    }
                ]
            };
            req.open('POST', url, false);
            req.setRequestHeader('Content-Type', "application/json")
            var data = JSON.stringify(query);
            req.send(data);
        });
        totalCount = parseInt(localStorage.getItem('counter'))
        if (totalCount <= 5) {
            document.getElementById("progress").src = `img/${totalCount}.png`;
        }
    }, false);
}, false);