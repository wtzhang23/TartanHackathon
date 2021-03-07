document.addEventListener('DOMContentLoaded', function() {
    var checkPageButton = document.getElementById('checkPage');
    checkPageButton.addEventListener('click', function() {
        chrome.tabs.executeScript(null, {
            code: `document.all[0].innerText`,
            allFrames: false, // this is the default
            runAt: 'document_start', // default is document_idle. See https://stackoverflow.com/q/42509273 for more details.
        }, function(results) {
            // results.length must be 1
            var result = results[0];
            console.log(result);
        });
    }, false);
  }, false);