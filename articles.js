var fs = require('fs');
const NewsAPI = require('newsapi');
const newsapi = new NewsAPI('6aa91d34573842b88c17af61f04bbbd8');
const topics = ["Gun Control", "Abortion", "Religious Freedom", "Animal Rights", "Vaccines", "Privacy", "Capitalism", "Climate Change", "Evolution", "Marijuana", "Capital Punishment", "Marriage", "Immigration", "Trump", "Opioid", "Transgender", "Universal Basic Income", "White Supremacy", "Green New Deal", "BLM", "Cancel Culture", "College", "Israel", "AI", "Atheist", "Social Security", "Addiction", "Fracking", "Nuclear", "Unions", "Insurance", "Agriculture", "Women", "Cryptocurrency", "Biofuel", "Medicine", "Speech", "COVID", "Mexican Border", "Net Neutrality", "Extremism", "Terrorism", "Paris Climate Accord", "Misinformation", "Voter Fraud", "Tarriff", "Supreme Court", "Stimulus", "Police", "Autopilot", "Sex", "Stem Cells", "Mental Health", "Slavery", "Death Penalty", "Vegetarian", "Vaping", "Cuba", "Athletes", "Brexit", "Feminism", "Gentrification", "Kashmir", "Drug", "Patriot Act", "Islam", "Buddhism", "AIDS", "Gene Editing", "ADHD", "Dyslexic", "Eugenics", "Euthanasia", "LGBTQ", "Gay", "Wildfire", "Hurricane", "Conservative", "Liberal", "Socialism", "Oil", "Pollution", "Tax", "Piracy"];
for (let topic of topics) {
    console.log(topic);
    newsapi.v2.everything({
        q: topic,
        from: '2021-02-08',
        to: '2021-03-06',
        language: 'en',
        sortBy: 'relevancy',
        page: 1
      }).then(response => {
        console.log(response);
        fs.writeFile(`articles/${topic}.txt`, JSON.stringify(response), function(err) {
            if (err) {
                console.log(err);
            }
        });
        /*
          {
            status: "ok",
            articles: [...]
          }
        */
      });
      console.log("test");
}