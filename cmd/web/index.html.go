package main

const indexTemplate = `<!DOCTYPE html>
<html>
<head>
    <link rel="stylesheet" href="https://unpkg.com/purecss@1.0.0/build/pure-min.css" integrity="sha384-nn4HPE8lTHyVtfCBi5yW9d20FjT8BJwUXyWZT9InLYax14RDjBj46LmSztkmNP9w" crossorigin="anonymous" />
    <link rel="stylesheet" href="https://fonts.googleapis.com/css?family=Palanquin" />
    <link rel="stylesheet" href="https://stackpath.bootstrapcdn.com/font-awesome/4.7.0/css/font-awesome.min.css" />
    <link rel="stylesheet" href="stylesheets/style.css" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>Substitute-Bot</title>
</head>

<body>
    <div class="header">
        <h1>Substitute-Bot</h1>
        <h2>VIM-style search & replacement for Reddit comments</h2>
    </div>

    <div class="content">
        <p>Substitute-Bot is a combination of a bot for Reddit that provides VIM style search + replace functionality for comments and also a front end web component (what you're reading right now). Written in Go. Redis is utilized very lightly for keeping a running list of posted comments.</p>

        <p>Syntax is VIM-like - <code>s/SEARCH/REPLACE</code> or <code>s#SEARCH#REPLACE</code>. Post a reply to another comment with this syntax and this bot will process your request & post your requested replacement.</p>

        <div class="pure-g center-text" id="link-icon-row">
            <div class="pure-u-1-3">
                <a href="https://github.com/anirbanmu/substitute-bot-go"><i class="fa fa-5x fa-github" aria-hidden="true"></i></a>
            </div>
            <div class="pure-u-1-3">
                <a href="https://www.reddit.com/user/{{ .BotUsername }}"><i class="fa fa-5x fa-reddit" aria-hidden="true"></i></a>
            </div>
            <div class="pure-u-1-3">
                <a href="https://www.reddit.com/message/compose/?to={{ .BotUsername }}"><i class="fa fa-5x fa-envelope" aria-hidden="true"></i></a>
            </div>
        </div>

        <div class="replies">
            <h2>Recent replies from bot</h2>
            {{ range $r := .Replies }}
                {{ with $r }}
                    <div class="pure-g reply-row">
                        <div class="pure-u-4-5">
                            {{ .RenderSanitizedHtmlForTemplate }}
                        </div>
                        <div class="pure-u-1-5 reply-details">
                            <p>{{ .RenderCreatedDateForTemplate }}</p>
                            <p>
                                <span>Requested by</span>
                                <a href="https://www.reddit.com/u/{{ .Requester }}">/u/{{ .Requester }}</a>
                            </p>
                            <a href="https://www.reddit.com{{ .Permalink }}">Comment link</a>
                        </div>
                    </div>
                  {{ end }}
            {{ end }}
        </div>
    </div>
</body>
</html>
`
