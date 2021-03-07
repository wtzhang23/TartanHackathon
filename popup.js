var totalCount = 0;
document.addEventListener('DOMContentLoaded', function() {
    var checkPageButton = document.getElementById('checkPage');
    checkPageButton.addEventListener('click', function() {
        totalCount+=1;
        ///Gets the text from the page
        chrome.tabs.executeScript(null, {
            code: `document.all[0].innerText`,
            allFrames: false,
            runAt: 'document_start',
        }, function(results) {
            var result = results[0];
            console.log(result);
        });
        if (totalCount <=5) {
            document.getElementById("progress").src = `img/${totalCount}.png`;
        }
    }, false);
  }, false);