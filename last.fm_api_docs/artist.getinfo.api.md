---
created: 2026-06-26T23:53:00 (UTC -04:00)
tags: []
source: https://www.last.fm/api/show/album.getInfo
author: 
---

# API Docs | Last.fm

> ## Excerpt
> The world's largest online music service. Listen online, find out more about your favourite artists, and get music recommendations, only at Last.fm

---
## [#](https://www.last.fm/api/show/album.getInfo#album-getinfo) album.getInfo

Get the metadata and tracklist for an album on Last.fm using the album name or a musicbrainz id.

## [#](https://www.last.fm/api/show/album.getInfo#example-urls) Example URLs

**JSON:** [/2.0/?method=album.getinfo&api\_key=YOUR\_API\_KEY&artist=Cher&album=Believe&format=json (opens new window)](http://ws.audioscrobbler.com/2.0/?method=album.getinfo&api_key=YOUR_API_KEY&artist=Cher&album=Believe&format=json)  
**XML:** [/2.0/?method=album.getinfo&api\_key=YOUR\_API\_KEY&artist=Cher&album=Believe (opens new window)](http://ws.audioscrobbler.com/2.0/?method=album.getinfo&api_key=YOUR_API_KEY&artist=Cher&album=Believe)

## [#](https://www.last.fm/api/show/album.getInfo#params) Params

**artist** (Required (unless mbid)\] : The artist name  
**album** (Required (unless mbid)\] : The album name  
**mbid** (Optional) : The musicbrainz id for the album  
**autocorrect\[0|1\]** (Optional) : Transform misspelled artist names into correct artist names, returning the correct version instead. The corrected artist name will be returned in the response.  
**username** (Optional) : The username for the context of the request. If supplied, the user's playcount for this album is included in the response.  
**lang** (Optional) : The language to return the biography in, expressed as an ISO 639 alpha-2 code.  
**api\_key** (Required) : A Last.fm API key.

## [#](https://www.last.fm/api/show/album.getInfo#auth) Auth

This service does **not** require authentication.

## [#](https://www.last.fm/api/show/album.getInfo#sample-response) Sample Response

## [#](https://www.last.fm/api/show/album.getInfo#attributes) Attributes

-   **duration** : In seconds

## [#](https://www.last.fm/api/show/album.getInfo#errors) Errors

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
