import pytest

from invoices import list_invoices, export_invoice


@pytest.mark.verifies("ABC-100.AC1~1")
def test_invoices_are_listed_newest_first():
    invoices = list_invoices(customer_id=7)
    assert [i.number for i in invoices] == ["INV-3", "INV-2", "INV-1"]


@pytest.mark.verifies("ABC-101.AC1~1")
def test_export_is_named_after_the_invoice_number():
    export = export_invoice("INV-3")
    assert export.filename == "INV-3.pdf"
    assert export.content_type == "application/pdf"
