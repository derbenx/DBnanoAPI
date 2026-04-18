const { chromium } = require('playwright');
const fs = require('fs');
const path = require('path');

(async () => {
  const browser = await chromium.launch();
  const page = await browser.newPage();

  // Mock Wails runtime and backend
  await page.addInitScript(() => {
    window.go = {
      main: {
        App: {
          GetConfig: () => Promise.resolve({
            APIKeyPaid: "paid-key",
            APIKeyFree: "free-key",
            IsFreeModeImage: true,
            IsFreeModeChat: false,
            ChatModelList: "model1\nmodel2",
            Temperature: 0.7,
            TopP: 0.9,
            TopK: 40,
            MaxOutputTokens: 2048
          }),
          GetImages: () => Promise.resolve([]),
          GetTasks: () => Promise.resolve([]),
          GetBatches: () => Promise.resolve([]),
          CalculateChatCost: () => Promise.resolve(0.0012),
          GetLogs: () => Promise.resolve("System initialized...")
        }
      }
    };
    window.runtime = {
      EventsOn: () => {},
      LogInfo: (msg) => console.log('WailsLog:', msg)
    };
  });

  const htmlPath = 'file://' + path.resolve('frontend/index.html');
  await page.goto(htmlPath);

  // Wait for initial load
  await page.waitForTimeout(1000);

  // Test Image Tab Toggles
  console.log('Verifying Image Tab...');
  const freeModeCheckbox = await page.$('#free-mode-image');
  if (freeModeCheckbox) {
      console.log('Found Image Free Mode checkbox');
  }
  await page.screenshot({ path: '/home/jules/verification/create_tab_reverify.png' });

  // Test Settings Tab
  console.log('Navigating to Settings Tab...');
  await page.evaluate(() => showTab('settings-tab'));
  await page.waitForTimeout(500);
  await page.screenshot({ path: '/home/jules/verification/settings_tab_reverify.png' });

  // Check for expected fields in Settings
  const paidKey = await page.$eval('#api-key-paid', el => el.value).catch(() => 'NOT FOUND');
  console.log('Paid Key in UI:', paidKey);

  // Test Chat Tab
  console.log('Navigating to Chat Tab...');
  await page.evaluate(() => showTab('chat-tab'));
  await page.waitForTimeout(500);
  await page.screenshot({ path: '/home/jules/verification/chat_tab_reverify.png' });

  const chatInput = await page.$('#chat-input');
  if (chatInput) console.log('Found Chat input');

  await browser.close();
})();
