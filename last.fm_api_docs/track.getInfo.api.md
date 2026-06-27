---
created: 2026-06-26T23:54:42 (UTC -04:00)
tags: []
source: https://www.last.fm/api/show/track.getInfo
author: 
---

# API Docs | Last.fm

> ## Excerpt
> The world's largest online music service. Listen online, find out more about your favourite artists, and get music recommendations, only at Last.fm

---
## [#](https://www.last.fm/api/show/track.getInfo#track-getinfo) track.getInfo

Get the metadata for a track on Last.fm using the artist/track name or a musicbrainz id.

## [#](https://www.last.fm/api/show/track.getInfo#example-urls) Example URLs

**JSON:** [/2.0/?method=track.getInfo&api\_key=YOUR\_API\_KEY&artist=cher&track=believe&format=json (opens new window)](http://ws.audioscrobbler.com/2.0/?method=track.getInfo&api_key=YOUR_API_KEY&artist=cher&track=believe&format=json)  
**XML:** [/2.0/?method=track.getInfo&api\_key=YOUR\_API\_KEY&artist=cher&track=believe (opens new window)](http://ws.audioscrobbler.com/2.0/?method=track.getInfo&api_key=YOUR_API_KEY&artist=cher&track=believe)

## [#](https://www.last.fm/api/show/track.getInfo#params) Params

**mbid** (Optional) : The musicbrainz id for the track  
**track** (Required (unless mbid)\] : The track name  
**artist** (Required (unless mbid)\] : The artist name  
**username** (Optional) : The username for the context of the request. If supplied, the user's playcount for this track and whether they have loved the track is included in the response.  
**autocorrect\[0|1\]** (Optional) : Transform misspelled artist and track names into correct artist and track names, returning the correct version instead. The corrected artist and track name will be returned in the response.  
**api\_key** (Required) : A Last.fm API key.

## [#](https://www.last.fm/api/show/track.getInfo#auth) Auth

This service does **not** require authentication.

## [#](https://www.last.fm/api/show/track.getInfo#sample-response) Sample Response

## [#](https://www.last.fm/api/show/track.getInfo#attributes) Attributes

-   **duration** : In milliseconds
-   **fulltrack** : An attribute value of 1 indicates a full length preview is available for streaming
-   **streamable** : A tag value of 1 indicates a 30 second preview of this song is available for streaming

## [#](https://www.last.fm/api/show/track.getInfo#errors) Errors

-   **2** : Invalid service - This service does not exist
-   **3** : Invalid Method - No method with that name in this package
-   **4** : Authentication Failed - You do not have permissions to access the service
-   **5** : Invalid format - This service doesn't exist in that format
-   **6** : Invalid parameters - Your request is missing a required parameter
-   **7** : Invalid resource specified
-   **8** : Operation failed - Something else went wrong
-   **9** : Invalid session key - Please re-authenticate
-   **10** : Invalid API key - You must be granted a valid key by last.fm
-   **11** : Service Offline - This service is temporarily offline. Try again later.
-   **13** : Invalid method signature supplied
-   **16** : There was a temporary error processing your request. Please try again
-   **26** : Suspended API key - Access for your account has been suspended, please contact Last.fm
-   **29** : Rate limit exceeded - Your IP has made too many requests in a short period
