(() => {
    $(() => {
        $('.card .card-body').each((idx, el) => {
            const $el = $(el);
            if ($el.text().trim().length === 0) {
                $el.addClass('card-body-collapse');
            }
        });
    });
})();
