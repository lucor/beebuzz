# Browser Support

[BeeBuzz Hive](https://hive.beebuzz.app) is a [PWA](#what-is-a-pwa) that needs a browser with [Web Push](https://developer.mozilla.org/en-US/docs/Web/API/Push_API) and [Web Crypto](https://developer.mozilla.org/en-US/docs/Web/API/Web_Crypto_API) `X25519` support for secure device registration.

The minimum supported version is whichever requirement is higher on your platform.

## Supported Browsers

| Browser / Platform   | Minimum | Installation  | Notes                                                                                                                                                                                        |
| -------------------- | ------- | ------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Chrome               | 133+    | Recommended   | Most reliable when installed                                                                                                                                                                 |
| Edge                 | 133+    | Recommended   | Most reliable when installed                                                                                                                                                                 |
| Firefox              | 130+    | Not available | Firefox [does not support PWA installation](https://developer.mozilla.org/en-US/docs/Web/Progressive_web_apps/Guides/Making_PWAs_installable#browser_support); works as a browser-tab client |
| Safari (macOS)       | 17.0+   | Recommended   | Installation improves reliability                                                                                                                                                            |
| Safari (iPhone/iPad) | 17.0+   | Required      | Web Push only works from the Home Screen installed app                                                                                                                                       |
| Samsung Internet     | 29+     | Recommended   | Most reliable when installed                                                                                                                                                                 |

Other Chromium-based browsers may work if they provide the required Web Push and Web Crypto capabilities, but they are not officially tested.

Install BeeBuzz Hive when your browser offers it for the most reliable notifications and background behavior. On iPhone and iPad, installation is required.

## What Is A PWA?

A Progressive Web App runs from the browser and can be installed where the platform supports it, without going through an app store.

Learn more:

- [Progressive Web Apps on MDN](https://developer.mozilla.org/en-US/docs/Web/Progressive_web_apps)
- [Learn PWA on web.dev](https://web.dev/learn/pwa)

## Related

- [Quickstart](/docs/quickstart)
- [Local Dev](/docs/local-dev)
