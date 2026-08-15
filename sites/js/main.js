/**
 * Devo Official Website - Main JavaScript
 * Handles navigation, scroll animations, and mobile menu
 */

(function () {
    'use strict';

    // ============================================
    // i18n - Internationalization (must run FIRST)
    // ============================================
    const SUPPORTED_LANGS = ['zh-CN', 'en-US'];
    const DEFAULT_LANG = 'en-US';

    function detectInitialLang() {
        const urlParams = new URLSearchParams(window.location.search);
        const urlLang = urlParams.get('lang');
        if (urlLang && SUPPORTED_LANGS.includes(urlLang)) return urlLang;

        const saved = localStorage.getItem('lang');
        if (saved && SUPPORTED_LANGS.includes(saved)) return saved;

        return DEFAULT_LANG;
    }

    function t(key, fallback) {
        const dict = window.I18N && window.I18N[window.__CURRENT_LANG];
        if (dict && typeof dict[key] === 'string') return dict[key];
        return typeof fallback === 'string' ? fallback : key;
    }

    function applyI18n() {
        const lang = window.__CURRENT_LANG;
        const dict = window.I18N && window.I18N[lang];
        if (!dict) return;

        document.documentElement.setAttribute('lang', lang);
        document.documentElement.setAttribute('data-lang', lang);

        const ogLocale = lang === 'zh-CN' ? 'zh_CN' : 'en_US';
        const ogLocaleMeta = document.querySelector('meta[property="og:locale"]');
        if (ogLocaleMeta) ogLocaleMeta.setAttribute('content', ogLocale);

        // 1) Translate DOM elements with data-i18n (textContent)
        document.querySelectorAll('[data-i18n]').forEach(function (el) {
            const key = el.getAttribute('data-i18n');
            if (dict[key] !== undefined) {
                const val = dict[key];
                if (val.indexOf('<') >= 0 || val.indexOf('&') >= 0) {
                    el.innerHTML = val;
                } else {
                    el.textContent = val;
                }
            }
        });

        // 2) Translate <title>
        if (dict['meta.title']) document.title = dict['meta.title'];

        // 3) Translate meta[name="description"], meta[name="keywords"]
        const descMeta = document.querySelector('meta[name="description"]');
        if (descMeta && dict['meta.description']) descMeta.setAttribute('content', dict['meta.description']);
        const kwMeta = document.querySelector('meta[name="keywords"]');
        if (kwMeta && dict['meta.keywords']) kwMeta.setAttribute('content', dict['meta.keywords']);

        // 4) Translate og:title / og:description / twitter versions
        ['og:title', 'og:description'].forEach(function (prop) {
            const key = prop.replace('og:', 'og.');
            const els = document.querySelectorAll('meta[property="' + prop + '"], meta[name="' + prop + '"]');
            els.forEach(function (m) {
                if (dict[key]) m.setAttribute('content', dict[key]);
            });
        });

        // 5) Update FAQ JSON-LD (if exists)
        const faqScript = document.getElementById('ld-json-faq');
        if (faqScript) {
            try {
                const faqData = {
                    '@context': 'https://schema.org',
                    '@type': 'FAQPage',
                    'mainEntity': [
                        {
                            '@type': 'Question',
                            'name': dict['faq.q1'],
                            'acceptedAnswer': {
                                '@type': 'Answer',
                                'text': dict['faq.a1']
                            }
                        },
                        {
                            '@type': 'Question',
                            'name': dict['faq.q2'],
                            'acceptedAnswer': {
                                '@type': 'Answer',
                                'text': dict['faq.a2']
                            }
                        }
                    ]
                };
                faqScript.textContent = JSON.stringify(faqData);
            } catch (e) { /* ignore */ }
        }

        // 6) Update SoftwareApplication JSON-LD description
        const appScript = document.getElementById('ld-json-app');
        if (appScript) {
            try {
                const parsed = JSON.parse(appScript.textContent);
                if (dict['ldjson.description']) parsed.description = dict['ldjson.description'];
                if (dict['ldjson.name']) parsed.name = dict['ldjson.name'];
                appScript.textContent = JSON.stringify(parsed);
            } catch (e) { /* ignore */ }
        }
    }

    function switchLang(newLang) {
        if (!SUPPORTED_LANGS.includes(newLang)) return;
        window.__CURRENT_LANG = newLang;
        localStorage.setItem('lang', newLang);
        applyI18n();

        const langToggleText = document.querySelector('#langToggle span');
        if (langToggleText) {
            langToggleText.textContent = newLang === 'zh-CN' ? 'English' : '中文';
        }

        if (typeof window.hljsUpdateTermText === 'function') {
            window.hljsUpdateTermText();
        }
    }

    // Init i18n immediately
    window.__CURRENT_LANG = detectInitialLang();

    function initDomI18n() {
        applyI18n();
        // bind switch button
        const langBtn = document.getElementById('langToggle');
        if (langBtn) {
            langBtn.addEventListener('click', function () {
                const next = window.__CURRENT_LANG === 'zh-CN' ? 'en-US' : 'zh-CN';
                switchLang(next);
            });
        }
        // update lang button label
        const langToggleText = document.querySelector('#langToggle span');
        if (langToggleText) {
            langToggleText.textContent = window.__CURRENT_LANG === 'zh-CN' ? 'English' : '中文';
        }
    }

    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', initDomI18n);
    } else {
        initDomI18n();
    }

    // expose for testing
    window.__switchLang = switchLang;
    window.__i18nT = t;

    // ============================================
    // Mobile Navigation Toggle
    // ============================================
    const navToggle = document.querySelector('.nav-toggle');
    const navLinks = document.querySelector('.nav-links');

    if (navToggle && navLinks) {
        navToggle.addEventListener('click', function () {
            const isActive = navLinks.classList.toggle('active');
            navToggle.classList.toggle('active');
            navToggle.setAttribute('aria-expanded', isActive);
        });

        // Close menu when clicking a link
        navLinks.querySelectorAll('a').forEach(function (link) {
            link.addEventListener('click', function () {
                navLinks.classList.remove('active');
                navToggle.classList.remove('active');
                navToggle.setAttribute('aria-expanded', 'false');
            });
        });

        // Close menu when clicking outside
        document.addEventListener('click', function (e) {
            if (!navToggle.contains(e.target) && !navLinks.contains(e.target)) {
                navLinks.classList.remove('active');
                navToggle.classList.remove('active');
                navToggle.setAttribute('aria-expanded', 'false');
            }
        });
    }

    // ============================================
    // Scroll-based Navigation Background
    // ============================================
    const nav = document.getElementById('nav');
    let lastScrollY = 0;

    function updateNav() {
        const scrollY = window.scrollY;
        const style = getComputedStyle(document.documentElement);
        if (scrollY > 50) {
            nav.style.background = style.getPropertyValue('--nav-bg-scrolled').trim();
        } else {
            nav.style.background = style.getPropertyValue('--nav-bg').trim();
        }
        lastScrollY = scrollY;
    }

    // Throttle scroll handler
    let scrollTicking = false;
    window.addEventListener('scroll', function () {
        if (!scrollTicking) {
            window.requestAnimationFrame(function () {
                updateNav();
                scrollTicking = false;
            });
            scrollTicking = true;
        }
    }, { passive: true });

    // ============================================
    // Intersection Observer for Fade-in Animations
    // ============================================
    const fadeElements = document.querySelectorAll(
        '.feature-card, .workflow-step, .security-item, .architecture-layer, .architecture-highlight, .interface-feature, .roadmap-item, .contact-card'
    );

    if ('IntersectionObserver' in window) {
        const observer = new IntersectionObserver(
            function (entries) {
                entries.forEach(function (entry) {
                    if (entry.isIntersecting) {
                        entry.target.classList.add('visible');
                        observer.unobserve(entry.target);
                    }
                });
            },
            {
                threshold: 0.1,
                rootMargin: '0px 0px -50px 0px'
            }
        );

        fadeElements.forEach(function (el) {
            el.classList.add('fade-in');
            observer.observe(el);
        });
    } else {
        // Fallback for browsers without IntersectionObserver
        fadeElements.forEach(function (el) {
            el.classList.add('visible');
        });
    }

    // ============================================
    // Smooth scroll for anchor links (fallback)
    // ============================================
    document.querySelectorAll('a[href^="#"]').forEach(function (anchor) {
        anchor.addEventListener('click', function (e) {
            const targetId = this.getAttribute('href');
            if (targetId === '#') return;

            const target = document.querySelector(targetId);
            if (target) {
                e.preventDefault();
                const navHeight = parseInt(getComputedStyle(document.documentElement).getPropertyValue('--nav-height')) || 64;
                const targetPosition = target.getBoundingClientRect().top + window.scrollY - navHeight;

                window.scrollTo({
                    top: targetPosition,
                    behavior: 'smooth'
                });
            }
        });
    });

    // ============================================
    // Active Navigation Link Highlighting
    // ============================================
    const sections = document.querySelectorAll('section[id]');
    const navAnchors = document.querySelectorAll('.nav-links a[href^="#"]');

    function highlightNav() {
        const scrollY = window.scrollY + 100;

        sections.forEach(function (section) {
            const sectionTop = section.offsetTop;
            const sectionHeight = section.offsetHeight;
            const sectionId = section.getAttribute('id');

            if (scrollY >= sectionTop && scrollY < sectionTop + sectionHeight) {
                navAnchors.forEach(function (a) {
                    a.style.color = '';
                    if (a.getAttribute('href') === '#' + sectionId) {
                        a.style.color = 'var(--color-primary)';
                    }
                });
            }
        });
    }

    window.addEventListener('scroll', highlightNav, { passive: true });

    // ============================================
    // Terminal Typing Animation (optional enhancement)
    // ============================================
    const terminalLines = document.querySelectorAll('.terminal-line');
    if (terminalLines.length > 0) {
        // Show lines with staggered delay
        terminalLines.forEach(function (line, index) {
            line.style.opacity = '0';
            line.style.transform = 'translateY(8px)';
            line.style.transition = 'opacity 0.3s ease, transform 0.3s ease';

            setTimeout(function () {
                line.style.opacity = '1';
                line.style.transform = 'translateY(0)';
            }, 300 + index * 200);
        });
    }

    // ============================================
    // Performance: Reduce animations for low-end devices
    // ============================================
    if (navigator.hardwareConcurrency && navigator.hardwareConcurrency <= 2) {
        // Disable complex animations on low-end devices
        document.querySelectorAll('.fade-in').forEach(function (el) {
            el.classList.add('visible');
            el.style.transition = 'none';
        });
    }

    // ============================================
    // Theme Toggle (Dark / Light)
    // ============================================
    const themeToggle = document.querySelector('.theme-toggle');
    const html = document.documentElement;

    function getSystemTheme() {
        return window.matchMedia('(prefers-color-scheme: light)').matches ? 'light' : 'dark';
    }

    function setTheme(theme) {
        html.setAttribute('data-theme', theme);
        localStorage.setItem('theme', theme);
        updateNav();
    }

    function toggleTheme() {
        const current = html.getAttribute('data-theme') || 'dark';
        setTheme(current === 'dark' ? 'light' : 'dark');
    }

    // Init theme from localStorage or system preference
    (function initTheme() {
        const saved = localStorage.getItem('theme');
        if (saved) {
            setTheme(saved);
        } else {
            setTheme(getSystemTheme());
        }
    })();

    if (themeToggle) {
        themeToggle.addEventListener('click', toggleTheme);
    }

    // Listen for system theme changes when no manual preference is set
    window.matchMedia('(prefers-color-scheme: light)').addEventListener('change', function (e) {
        if (!localStorage.getItem('theme')) {
            setTheme(e.matches ? 'light' : 'dark');
        }
    });

    // ============================================
    // Quick Start Tabs Switching
    // ============================================
    const quickstartTabs = document.querySelectorAll('.quickstart-tab-btn');
    const quickstartContents = document.querySelectorAll('.quickstart-tab-content');

    if (quickstartTabs.length > 0 && quickstartContents.length > 0) {
        quickstartTabs.forEach(function (tab) {
            tab.addEventListener('click', function () {
                const targetTab = this.getAttribute('data-tab');

                // Remove active class from all tabs and contents
                quickstartTabs.forEach(function (t) { t.classList.remove('active'); });
                quickstartContents.forEach(function (c) { c.classList.remove('active'); });

                // Add active class to clicked tab and matching content
                this.classList.add('active');
                const targetContent = document.querySelector('[data-tab-content="' + targetTab + '"]');
                if (targetContent) {
                    targetContent.classList.add('active');
                }
            });
        });
    }

    // ============================================
    // FAQ Accordion Enhancement
    // ============================================
    const faqItems = document.querySelectorAll('.faq-item');
    if (faqItems.length > 0) {
        faqItems.forEach(function (item) {
            const question = item.querySelector('.faq-question');
            if (question) {
                question.addEventListener('keydown', function (e) {
                    if (e.key === 'Enter' || e.key === ' ') {
                        e.preventDefault();
                        if (item.hasAttribute('open')) {
                            item.removeAttribute('open');
                        } else {
                            item.setAttribute('open', '');
                        }
                    }
                });
            }
        });
    }

    // ============================================
    // Fade-in: Add more selectors
    // ============================================
    const moreFadeElements = document.querySelectorAll(
        '.mode-card, .tech-layer, .arch-row, .quickstart-step, .faq-item'
    );

    if ('IntersectionObserver' in window) {
        const additionalObserver = new IntersectionObserver(
            function (entries) {
                entries.forEach(function (entry) {
                    if (entry.isIntersecting) {
                        entry.target.classList.add('visible');
                        additionalObserver.unobserve(entry.target);
                    }
                });
            },
            {
                threshold: 0.1,
                rootMargin: '0px 0px -40px 0px'
            }
        );

        moreFadeElements.forEach(function (el) {
            el.classList.add('fade-in');
            additionalObserver.observe(el);
        });
    } else {
        moreFadeElements.forEach(function (el) {
            el.classList.add('visible');
        });
    }

})();