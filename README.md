# Allbin
<sub><sup>logos are bloat</sup></sub>

# What is it?

Allbin is a free and open-source pastebin using a stateless golang backend, essentially a dumb vault. Data is encrypted on the client side browser using AES-256 bit security.
"Dimensions" can be either public or private and can contain a title, a file and some <span style="color:#FFFFFF">text</span>.
<span style="color:#0F40A3">Encryption</span> involves the file, the <span style="color:#F7F73B">file name</span> and the <span style="color:#FFFFFF">text</span>. Assuming its encrypted at all. A <span style="color:#CC28D1">download limit</span> can be set for the dimension.

A typical dimension stored in the vault looks like this:

| Field | Type   | Description                                                                            |
| ---   | ---    | ---                                                                                    |
| <span style="color:#FF5E3D">title</span>          | string  | Optional display title                                                       |
| <span style="color:#0F40A3">encrypted</span>       | boolean | indicates encryption status                                                  |
| <span style="color:#F7F73B">fileName</span>        | string  | empty if no file added to dimension, otherwise the name of the file uploaded |
| <span style="color:#FFFFFF">text</span>           | string  | The message of the dimension                                                 |
| <span style="color:#CC28D1">downloadLimit</span>    | <span style="color:#234862">int</span>       | The amount of times a dimension can be downloaded/viewed                     |
| <span style="color:#23A64B">reads</span>          | <span style="color:#234862">int</span>       | The amount of times a dimension has been viewed                              |
| <span style="color:#9E1C02">expirationDate</span> | <span style="color:#234862">int</span>     | The expiration date of the dimension in epoch time                           |

# Additional packet info
Dimensions are stored in json format.
Users can add a <span style="color:#FF5E3D">title</span>, a file, some <span style="color:#FFFFFF">text</span>, a <span style="color:#CC28D1">download limit</span> and an <span style="color:#9E1C02">expiration date</span>. The <span style="color:#FF5E3D">title</span> can be empty. It's purely for display. <br>
The <span style="color:#9E1C02">expiration date</span> can't be longer than 3 months. This is verfied by the server. <br>
<span style="color:#CC28D1">Download limit</span> is unlimited if less than 1. <br>


# Docker
