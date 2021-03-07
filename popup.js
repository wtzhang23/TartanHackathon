const req = new XMLHttpRequest();
const params = { "num": 3 };
const url = "https://tartanhackathon.uc.r.appspot.com/recommend?num=10";
function getText() {
    return document.body.innerText
}
if ("counter" in localStorage) {
    let today = new Date();
    if (localStorage['lastUpdate'] != null) {
        if (parseInt(localStorage['lastUpdate']) != today.getDate()) {
            console.log("DEBUG");
            localStorage['lastUpdate'] =  today.getDate();
            localStorage['counter'] = 0;
        }
    }
} else {
    localStorage['counter'] = 0;
    localStorage['lastUpdate'] =  new Date().getDate();
}
document.getElementById("progress").src = `img/${Math.min(5, localStorage['counter'])}.png`;
document.addEventListener('DOMContentLoaded', function () {
    var checkPageButton = document.getElementById('checkPage');
    checkPageButton.addEventListener('click', function (e) {
        var count = parseInt(localStorage['counter']);
        localStorage.setItem('counter', count + 1);
        e.preventDefault();
        req.onload = function() {
            var articles = JSON.parse(req.responseText)
            //jswin = window.open("", "jswin", "width=550,height=450");
            for (article of articles["urls"]) {
                //jswin.document.write(article + "\n");
                chrome.tabs.create({'url': article, 'active': false}, function(tab){})
            }
        }
        
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
            req.open('POST', url, true);
            req.setRequestHeader('Content-Type', "application/json")
            var data = JSON.stringify(query);
            req.send(data);
        });
        totalCount = parseInt(localStorage.getItem('counter'))
        if (totalCount <= 5) {
            document.getElementById("progress").src = `img/${Math.min(5, localStorage['counter'])}.png`;
        }
    }, false);
}, false);