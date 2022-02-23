module.exports = {
    env: {
        node: true,
        es6: true,
        jest: true,
    },
    extends: [
        'eslint:recommended',
        'plugin:@typescript-eslint/eslint-recommended',
        // 'plugin:@typescript-eslint/recommended-requiring-type-checking',
    ],
    plugins: ['@typescript-eslint'],
    globals: { JSX: true, NodeJS: true },
    parser: '@typescript-eslint/parser',
    parserOptions: {
        ecmaFeatures: {
            jsx: true,
        },
        ecmaVersion: 6,
        project: './tsconfig.json',
        projectFolderIgnoreList: [],
        sourceType: 'module',
        // eslint-disable-next-line @typescript-eslint/no-unsafe-assignment
        tsconfigRootDir: __dirname,
    },
    rules: {
        'arrow-parens': ['error', 'always'],
        'block-spacing': ['error', 'always'],
        'brace-style': ['error', '1tbs', { allowSingleLine: true }],
        'comma-dangle': ['error', 'always-multiline'],
        'constructor-super': 'warn',
        'max-len': ['error', { code: 120, tabWidth: 4 }],
        'no-const-assign': 'warn',
        'no-this-before-super': 'warn',
        'no-undef': 'warn',
        'no-unreachable': 'warn',
        'no-unused-vars': 'off',
        'no-var': 'warn',
        'object-curly-spacing': ['error', 'always'],
        quotes: ['error', 'single', { avoidEscape: true }],
        semi: ['error', 'always'],
        'space-after-keywords': 'off',
        'space-before-blocks': [
            'error',
            { functions: 'always', keywords: 'always', classes: 'always' },
        ],
        'space-infix-ops': ['error', { int32Hint: false }],
        'keyword-spacing': [
            'error',
            {
                before: true,
                after: true,
                overrides: { catch: { after: true } },
            },
        ],
        'valid-typeof': 'warn',
        '@typescript-eslint/explicit-member-accessibility': 'off',
        "@typescript-eslint/no-unused-vars": "error",
    },
    overrides: [
        {
            // enable the rule specifically for TypeScript files
            files: ['*.ts', '*.tsx'],
            rules: {
                '@typescript-eslint/explicit-member-accessibility': ['error'],
            },
        },
    ],
};
